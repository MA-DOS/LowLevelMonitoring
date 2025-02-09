package main

import (
	"fmt"

	"github.com/MA-DOS/LowLevelMonitoring/client"
	"github.com/sirupsen/logrus"
)

const configFilePath = "config.yml"

func main() {
	// Load the configuration file.
	config, err := client.NewConfig(configFilePath)
	if err != nil {
		logrus.Error("Error reading config file: ", err)
		fmt.Print(config)
		return
	}

	// Get a running workflow.
	// workflowActive := watcher.GetWorkflowRun("../SlurmSetup/nextflow/chipseq/.nextflow.log")

	// Get the monitoring interval from the configuration file.
	monitoringInterval := config.ServerConfigurations.Prometheus.TargetServer.FetchInterval

	// if workflowActive {
	// Start the monitoring loop.
	client.ScheduleMonitoring(*config, configFilePath, monitoringInterval)
	// }

}
