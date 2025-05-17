package client

import (
	"encoding/json"
	"fmt"
	"net"
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
	Address       string   `yaml:"address"`
	Timeout       string   `yaml:"timeout"`
	FetchInterval int      `yaml:"interval"`
	Workers       []string `yaml:"workers"`
	Controller    string   `yaml:"controller"`
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

func CreateTCPListener(controllerIP string) (*net.TCPListener, error) {
	ip := net.ParseIP(controllerIP)
	if ip == nil {
		logrus.Errorf("Invalid IP address: %s", controllerIP)
		return nil, fmt.Errorf("invalid IP address: %s", controllerIP)
	}

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   ip,
		Port: 42,
	})
	if err != nil {
		logrus.Errorf("Error creating TCP listener on %s:%d:", controllerIP, err)
		return nil, err
	}

	logrus.Infof("Listening for container events on %s", listener.Addr())
	return listener, nil
}

func ListenForContainerEvents(c *Config, configPath string, containerEventChannel chan<- watcher.NextflowContainer) {
	// Create the TCP listener using the helper function.
	listener, err := CreateTCPListener(c.ServerConfigurations.Prometheus.TargetServer.Controller)
	if err != nil {
		return
	}

	// Run the listener in a goroutine to keep it active.
	go func(listener *net.TCPListener) {
		defer listener.Close() // Ensure the listener is closed when the goroutine exits.
		for {
			conn, err := listener.Accept()
			if err != nil {
				logrus.Errorf("Error accepting connection: %v", err)
				continue
			}
			logrus.Infof("Accepted connection from %s", conn.RemoteAddr())

			// Handle the connection in a separate goroutine.
			go HandleIncomingContainerEvents(conn, containerEventChannel)
		}
	}(listener)
}

func HandleIncomingContainerEvents(con net.Conn, containerEventChannel chan<- watcher.NextflowContainer) {
	defer con.Close()

	// Read the incoming data from the connection.
	buffer := make([]byte, 1024)
	n, err := con.Read(buffer)
	if err != nil {
		logrus.Error("Error reading from connection: ", err)
		return
	}

	// Deserialize the incoming data into a NextflowContainer.
	var container watcher.NextflowContainer
	err = json.Unmarshal(buffer[:n], &container)
	if err != nil {
		logrus.Error("Error deserializing container data: ", err)
		return
	}

	// Handle the event based on its type.
	switch container.ContainerEvent {
	case "[STARTED]":
		logrus.Infof("[REMOTE START EVENT] Writing container %s to output.", container.Name)
		// watcher.WriteToOutput(container) // Write the container data to output.
		watcher.WriteStartedToOutput(container) // Write the container data to output.
	case "[DIED]":
		logrus.Info("[REMOTE DIE EVENT] Writing container to output and monitoring channel.", container)
		containerEventChannel <- container   // Forward the container event to the monitoring logic.
		watcher.WriteDiedToOutput(container) // Write the container data to output.
	}
}

func WatchContainerEvents(containerEventChannel chan<- watcher.NextflowContainer) {
	workflowContainer := watcher.NextflowContainer{}
	go workflowContainer.GetContainerEvents(containerEventChannel)
}

// Refactor to pass a nxf container object
func StartMonitoring(c *Config, cfp string, workflowContainer watcher.NextflowContainer) (map[string]map[string]map[string]model.Matrix, map[string][]string, error) {
	queriesMap := ConsolidateQueries(ReadMonitoringConfiguration(cfp)) // Ignore labels

	resultMap, QueryMetaInfo, err := FetchMonitoringSources(c, workflowContainer, queriesMap)
	if err != nil {
		logrus.Error("Error fetching queries: ", err)
		return resultMap, QueryMetaInfo, err
	}
	return resultMap, QueryMetaInfo, err
}

func ScheduleMonitoring(config *Config, configPath string) {
	monitorIsIdle := false

	// Init the event-based polling for container events.
	containerEventChannel := make(chan watcher.NextflowContainer)

	// Start listening for remote container events.
	go ListenForContainerEvents(config, configPath, containerEventChannel)

	// Watch local container events.
	WatchContainerEvents(containerEventChannel)

	// Run the main monitoring loop by receiving container events.
	for {
		select {
		case workflowContainer := <-containerEventChannel:
			monitorIsIdle = false
			ProcessContainerEvent(config, configPath, workflowContainer)
		case <-time.After(10 * time.Second):
			HandleIdleState(&monitorIsIdle)
		}
	}
}

func ProcessContainerEvent(config *Config, configPath string, workflowContainer watcher.NextflowContainer) {
	logrus.Infof("[RECEIVED DEAD CONTAINER] Container Name coming from channel: %s who lived for %v and has PID %v.", workflowContainer.Name, workflowContainer.LifeTime, workflowContainer.PID)

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
