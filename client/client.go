package client

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/MA-DOS/LowLevelMonitoring/aggregate"
	"github.com/MA-DOS/LowLevelMonitoring/watcher"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Configurations structures.
type TargetServer struct {
	Address       string `yaml:"address"`
	Timeout       string `yaml:"timeout"`
	FetchInterval int    `yaml:"interval"`
}

type Prometheus struct {
	TargetServer TargetServer `yaml:"target_server"`
}

type ServerConfigurations struct {
	Prometheus Prometheus `yaml:"prometheus"`
	ConfigPath string     `yaml:"config_path"`
}
type Config struct {
	ServerConfigurations ServerConfigurations `yaml:"server_configurations"`
	MonitoringTargets    MonitoringTargets    `yaml:"monitoring_targets"`
}

type MonitoringTargets struct {
	TaskMetadata MonitoringTarget `yaml:"task_metadata"`
	CPU          MonitoringTarget `yaml:"cpu"`
	Memory       MonitoringTarget `yaml:"memory"`
	Disk         MonitoringTarget `yaml:"disk"`
	Network      MonitoringTarget `yaml:"network"`
	Energy       MonitoringTarget `yaml:"energy"`
}

type MonitoringTarget struct {
	Enabled     bool         `yaml:"enabled"`
	DataSources []DataSource `yaml:"metrics"`
}

type DataSource struct {
	Source    string   `yaml:"source"`
	Labels    []string `yaml:"labels"`
	Identfier string   `yaml:"identifier"`
	Metrics   []string `yaml:"metrics"`
}

type Metric struct {
	Name  string `yaml:"name"`
	Query string `yaml:"query"`
}

// LoadConfig loads the configuration from the file and returns implicit the prometheus configuration.
func NewConfig(configFilePath string) (*Config, error) {
	// Read in the config yml.
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		logrus.Error("Error creating config file: ", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		logrus.Error("Error unmarshalling the config file: ", err)
	}
	return &config, nil
}

func NewFetchClient(c *Config) (api.Client, error) {
	client, err := api.NewClient(api.Config{
		Address: c.ServerConfigurations.Prometheus.TargetServer.Address,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Using Prometheus API to fetch the monitoring targets.
func FetchMonitoringTargets(client api.Client, queryIdentifier, query string, containerStartUp, containerDie time.Time, containerName, cwdir, cid string, cpid int) (model.Matrix, error) {
	var wg sync.WaitGroup
	resultChannel := make(chan model.Matrix, 1) // Buffer to avoid blocking
	errorChannel := make(chan error, 1)         // Buffer for errors

	v1api := v1.NewAPI(client)

	// TODO: Use the queryIdentifier to fetch the query from the config dynamically.
	// Construct query for Nextflow container
	logrus.Info("Identifier is : ", queryIdentifier)
	logrus.Info("Query is : ", query)
	logrus.Info("Container Name is : ", containerName)
	logrus.Info("Container ID is : ", cid)
	logrus.Info("Container WorkDir is : ", cwdir)
	logrus.Info("Container PID is : ", cpid)

	nextflowQuery := func(query, queryIdentifier string) string {
		// TODO: Need to be distinguished depending on the identifier.
		switch queryIdentifier {
		// When name is identifier the container name can specify the query.
		case "name":
			return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, containerName)
		case "path":
			return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, cid)
		case "work_dir":
			return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, cwdir)
		case "groupname":
			return fmt.Sprintf(`%s{%s="%v"}`, query, queryIdentifier, cpid)
		case "container_names":
			return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, containerName)
		case "container_name":
			return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, containerName)
		}
		return query
	}

	logrus.Info("Querying Prometheus: ", nextflowQuery(query, queryIdentifier))

	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Perform range query
		result, warnings, err := v1api.QueryRange(ctx, nextflowQuery(query, queryIdentifier), v1.Range{
			Start: containerStartUp,
			End:   containerDie.Add(5 * time.Second), // Add seconds to ensure I get the last sample.
			Step:  500 * time.Millisecond,
		})

		if err != nil {
			logrus.Errorf("Error querying Prometheus: %v", err)
			errorChannel <- err
			logrus.Errorf("Error querying Prometheus: %v", err)
		}
		if len(warnings) > 0 {
			logrus.Warnf("Warnings: %v", warnings)
		}

		resultMatrix, ok := result.(model.Matrix)
		if !ok {
			logrus.Error("Error casting result to Matrix")
			errorChannel <- fmt.Errorf("failed to cast Prometheus response to Matrix")
			return
		}

		// logrus.Infof("[RESULT]: %v", resultMatrix)
		resultChannel <- resultMatrix
	}()

	// Separate goroutine to close channels after all work is done
	go func() {
		wg.Wait()
		close(resultChannel)
		close(errorChannel)
	}()

	// Wait for either a result or an error
	select {
	case result := <-resultChannel:
		return result, nil
	case err := <-errorChannel:
		return nil, err
	case <-time.After(12 * time.Second): // Timeout safeguard
		logrus.Error("Query timed out with no response")
		return nil, fmt.Errorf("prometheus query timeout")
	}
}

func ScheduleMonitoring(config Config, configPath string) {
	monitorIsIdle := false
	channelCounter := 0

	// Init the event-based polling for container events.
	containerEventChannel := make(chan watcher.NextflowContainer)
	workflowContainer := watcher.NextflowContainer{}
	go workflowContainer.GetContainerEvents(containerEventChannel)

	// Run the main monitoring loop by receiving container events.
	for {
		select {
		case workflowContainer := <-containerEventChannel:
			channelCounter++
			monitorIsIdle = false
			parsedStartTime, _ := time.Parse(time.RFC3339, workflowContainer.StartTime)
			parsedDieTime, _ := time.Parse(time.RFC3339, workflowContainer.DieTime)
			containerName := (workflowContainer.Name)
			lifeTime := parsedDieTime.Sub(parsedStartTime)
			containerWorkDir := workflowContainer.WorkDir
			containerID := workflowContainer.ContainerID
			containerPID := workflowContainer.PID

			logrus.Infof("[RECEIVED DEAD CONTAINER] Container Name coming from channel: %s who lived for %v.", containerName, lifeTime)

			// Run the Monitor against Prometheus.
			resultMap, queryMetaInfo, err := StartMonitoring(&config, configPath, parsedStartTime, parsedDieTime, containerName, containerWorkDir, containerID, containerPID)
			// fmt.Printf("[RESULT MAP]: %+v\n", resultMap)
			if err != nil {
				logrus.Error("Error starting monitoring: ", err)
				panic(err)
			}
			// Data Structure for results.
			for target, dataSources := range resultMap {
				for dataSource, queryNames := range dataSources {
					for queryName, samples := range queryNames {
						dataWrapper := aggregate.NewDataVectorWrapper(map[string]map[string]map[string]model.Matrix{
							target: {
								dataSource: {
									queryName: samples,
								},
							},
						}, queryMetaInfo)

						// Use the result variable
						// logrus.Infof("Processing result: %v", result)

						err = dataWrapper.CreateDataOutput()
						if err != nil {
							logrus.Error("Error creating output: ", err)
						}
					}
				}
			}
		case <-time.After(10 * time.Second):
			if !monitorIsIdle {
				logrus.Info("[WF MONITOR IDLE]")
				monitorIsIdle = true
			}
		}
	}
}
