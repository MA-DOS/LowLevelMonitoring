package client

import (
	"os"
	"time"

	"github.com/MA-DOS/LowLevelMonitoring/aggregate"
	"github.com/MA-DOS/LowLevelMonitoring/watcher"
	"github.com/prometheus/client_golang/api"
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

func HandleIdleState(monitorIsIdle *bool) {
	if !*monitorIsIdle {
		logrus.Info("[WF MONITOR IDLE]")
		*monitorIsIdle = true
	}
}

func WatchContainerEvents(containerEventChannel chan<- watcher.NextflowContainer) {
	workflowContainer := watcher.NextflowContainer{}
	go workflowContainer.GetContainerEvents(containerEventChannel)
}

// Refactor to pass a nxf container object
func StartMonitoring(c *Config, cfp string, workflowContainer watcher.NextflowContainer) (map[string]map[string]map[string]model.Matrix, map[string][]string, error) {
	queriesMap := ConsolidateQueries(ReadMonitoringConfiguration(cfp)) // Ignore labels

	// resultMap, QueryMetaInfo, err := FetchMonitoringSources(c, cst, cdt, cn, cwdir, cid, cpid, queriesMap)
	resultMap, QueryMetaInfo, err := FetchMonitoringSources(c, workflowContainer, queriesMap)
	if err != nil {
		logrus.Error("Error fetching queries: ", err)
		return resultMap, QueryMetaInfo, err
	}
	return resultMap, QueryMetaInfo, err
}

func ScheduleMonitoring(config Config, configPath string) {
	monitorIsIdle := false

	// Init the event-based polling for container events.
	containerEventChannel := make(chan watcher.NextflowContainer)
	WatchContainerEvents(containerEventChannel)

	// Run the main monitoring loop by receiving container events.
	for {
		select {
		case workflowContainer := <-containerEventChannel:
			monitorIsIdle = false
			ProcessContainerEvent(&config, configPath, workflowContainer)
		case <-time.After(10 * time.Second):
			HandleIdleState(&monitorIsIdle)
		}
	}
}

func ProcessContainerEvent(config *Config, configPath string, workflowContainer watcher.NextflowContainer) {
	logrus.Infof("[RECEIVED DEAD CONTAINER] Container Name coming from channel: %s who lived for %v.", workflowContainer.Name, workflowContainer.LifeTime)

	// Run the Monitor against Prometheus.
	resultMap, queryMetaInfo, err := StartMonitoring(config, configPath, workflowContainer)
	if err != nil {
		logrus.Error("Error starting monitoring: ", err)
		panic(err)
	}

	// Process results.
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
				if err := dataWrapper.CreateDataOutput(); err != nil {
					logrus.Error("Error creating output: ", err)
				}
			}
		}
	}
}
