package main

import (
	"fmt"

	"github.com/MA-DOS/LowLevelMonitoring/watcher"
)

// const configFilePath = "config.yml"

func main() {
	// Load the configuration file.
	// config, err := client.NewConfig(configFilePath)
	// if err != nil {
	// 	logrus.Error("Error reading config file: ", err)
	// 	fmt.Print(config)
	// 	return
	// }

	// Get the monitoring interval from the configuration file.
	// monitoringInterval := config.ServerConfigurations.Prometheus.TargetServer.FetchInterval

	// Start the monitoring loop.
	// client.ScheduleMonitoring(*config, configFilePath, monitoringInterval)

	// Test log file parsing.
	// taskFromLog := watcher.WorkflowTask{}
	// workflowActive := watcher.GetWorkflowRun("../SlurmSetup/nextflow/chipseq/.nextflow.log")
	// _ = workflowActive
	// _ = taskFromLog
	// if workflowActive {

	// 	err := taskFromLog.WatchCompletedTasks("/home/nfomin3/dev/SlurmSetup/nextflow/chipseq/.nextflow.log")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }

	// Test Docker API.
	nextflowContainer := watcher.NextflowContainer{}
	containers := nextflowContainer.ListContainers()
	fmt.Print("Inspect Output: ", nextflowContainer.InspectContainer(containers))
}
