package main

import (
	"time"

	"github.com/MA-DOS/LowLevelMonitoring/client"
	"github.com/sirupsen/logrus"
)

const configFilePath = "config/config.yml"

func main() {
	// Load the configuration file.
	config, err := client.NewConfig(configFilePath)
	if err != nil {
		logrus.Error("Error reading config file: ", err)
	}

	monitoringInterval := config.ServerConfigurations.Prometheus.TargetServer.FetchInterval

	// Start the monitoring loop.
	for {
		err := client.ScheduleMonitoring(config, configFilePath)
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Duration(monitoringInterval) * time.Second)
	}
}
