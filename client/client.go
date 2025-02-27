package client

import (
	"context"
	"fmt"
	"os"
	"time"

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
	Source  string   `yaml:"source"`
	Metrics []string `yaml:"metrics"`
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
func FetchMonitoringTargets(client api.Client, query string, containerStartUp, containerDie time.Time, containerName string) (model.Vector, error) {
	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use the current containers name to query only the relevant time series per job.
	nextflowQuery := func(query, containerName string) string {
		// cleanContainerName := watcher.EscapeContainerName(containerName)
		logrus.Info("Cleaned Container Name: ", containerName)
		return fmt.Sprintf(`%s{name=~"%s"}`, query, containerName)
	}

	logrus.Info("Querying Prometheus: ", nextflowQuery(query, containerName))

	// Range Query is used based on events of container engine.
	result, warnings, err := v1api.QueryRange(ctx, query, v1.Range{
		Start: containerStartUp,
		// TODO: Use time when container died here instead!!!
		// End:   time.Now(),
		End:  containerDie,
		Step: 1 * time.Second,
	})
	if err != nil {
		logrus.Errorf("Error querying Prometheus: %v", err)
		return nil, err
	}
	if len(warnings) > 0 {
		logrus.Warnf("Warnings: %v", warnings)
	}
	fmt.Println("Received Result from Prometheus: ", result)
	// Processing the received Matrix of SampleStreams into model.Vectors.
	// matrix := result.(model.Matrix)
	// vector := model.Vector{}
	// for _, sampleStream := range matrix {
	// 	for _, sample := range sampleStream.Values {
	// 		vector = append(vector, &model.Sample{
	// 			Metric:    sampleStream.Metric,
	// 			Value:     sample.Value,
	// 			Timestamp: sample.Timestamp,
	// 		})
	// }
	resultVector, ok := result.(model.Vector)
	if !ok {
		logrus.Error("Error casting to Vector")
		return nil, err
	}
	return resultVector, nil
}

// fmt.Println("Casted Vector: ", vector)

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
			_ = parsedStartTime
			_ = parsedDieTime
			// containerName := watcher.EscapeContainerName(workflowContainer.Name)
			containerName := (workflowContainer.Name)

			logrus.Info("[RECEIVED DEAD CONTAINER] Container Name coming from channel: ", containerName)
			logrus.Info("[CHANNEL COUNTER] Amount received through the channel: ", channelCounter)

			// Run the Monitor against Prometheus.
			// resultMap, err := StartMonitoring(&config, configPath, parsedStartTime, parsedDieTime, containerName)
			// fmt.Printf("Result Map: %+v\n", resultMap)
			// _ = resultMap
			// if err != nil {
			// 	logrus.Error("Error starting monitoring: ", err)
			// panic(err)
			// }
		// Data Structure for results.
		// for target, dataSources := range resultMap {
		// 	for dataSource, queryNames := range dataSources {
		// 		for queryName, samples := range queryNames {
		// 			dataWrapper := aggregate.NewDataVectorWrapper(map[string]map[string]map[string]model.Vector{
		// 				target: {
		// 					dataSource: {
		// 						queryName: samples,
		// 					},
		// 				},
		// 			})
		// 			// Use the result variable
		// 			// logrus.Infof("Processing result: %v", result)

		// 			err = dataWrapper.CreateDataOutput()
		// 			if err != nil {
		// 				logrus.Error("Error creating output: ", err)
		// 			}
		// 		}
		// 	}
		// }
		case <-time.After(10 * time.Second):
			if !monitorIsIdle {
				logrus.Info("[WF MONITOR IDLE] Container Engine probably busy...!")
				monitorIsIdle = true
			}

			// interval := config.ServerConfigurations.Prometheus.TargetServer.FetchInterval

			// // Sleep for the configured interval before polling again.
			// time.Sleep(time.Duration(interval) * time.Second)
		}
	}
}
