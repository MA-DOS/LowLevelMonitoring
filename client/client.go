package client

import (
	"context"
	"os"
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
func FetchMonitoringTargets(client api.Client, q string) (model.Vector, error) {
	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, warnings, err := v1api.Query(ctx, q, time.Now())
	if err != nil {
		logrus.Errorf("Error querying Prometheus: %v", err)
		return nil, err
	}
	if len(warnings) > 0 {
		logrus.Warnf("Warnings: %v", warnings)
	}
	castToVector, ok := result.(model.Vector)
	if !ok {
		logrus.Error("Error casting to Vector")
		return nil, err
	}
	return castToVector, nil
}

func ScheduleMonitoring(config Config, configPath string, interval int) {
	for {
		// Run the Monitor against Prometheus.
		resultMap, err := StartMonitoring(&config, configPath)
		if err != nil {
			logrus.Error("Error starting monitoring: ", err)
			// panic(err)
		}
		// Data Structure for results.
		for target, dataSources := range resultMap {
			for dataSource, queryNames := range dataSources {
				for queryName, samples := range queryNames {
					dataWrapper := aggregate.NewDataVectorWrapper(map[string]map[string]map[string]model.Vector{
						target: {
							dataSource: {
								queryName: samples,
							},
						},
					})
					// Use the result variable
					// logrus.Infof("Processing result: %v", result)

					err = dataWrapper.CreateDataOutput()
					if err != nil {
						logrus.Error("Error creating output: ", err)
					}
				}
			}
		}
		interval := config.ServerConfigurations.Prometheus.TargetServer.FetchInterval

		// Run the monitor against Docker Daemon.
		nextflowContainer := watcher.NextflowContainer{}
		containers := nextflowContainer.ListContainers()
		nextflowContainer.InspectContainer(containers)

		// Run the observer against the Nextflow log.
		// taskFromLog := watcher.WorkflowTask{}
		// taskFromLog.WatchCompletedTasks("/home/nfomin3/dev/SlurmSetup/nextflow/chipseq/.nextflow.log")

		// Sleep for the configured interval before polling again.
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
