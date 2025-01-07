package main

import (
	"fmt"
	"time"

	"github.com/MA-DOS/LowLevelMonitoring/aggregate"
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
	}

	// Get the monitoring interval from the configuration file.
	monitoringInterval := config.ServerConfigurations.Prometheus.TargetServer.FetchInterval

	// Start the monitoring loop.
	for {
		// Run the Monitor
		result, err := client.ScheduleMonitoring(config, configFilePath)
		if err != nil {
			panic(err)
		}

		err = aggregate.Write(&result)
		if err != nil {
			logrus.Error("Error writing results: ", err)
		}

		time.Sleep(time.Duration(monitoringInterval) * time.Second)
	}
}
