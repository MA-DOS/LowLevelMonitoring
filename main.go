package main

import (
	"github.com/MA-DOS/LowLevelMonitoring/client"
	"github.com/sirupsen/logrus"
)

const configFilePath = "config.yml"

func main() {
	// Load the configuration file.
	config, err := client.NewConfig(configFilePath)
	if err != nil {
		logrus.Error("Error reading config file: ", err)
		return
	}

	// Start the monitoring loop.
	client.ScheduleMonitoring(config, configFilePath)
}
