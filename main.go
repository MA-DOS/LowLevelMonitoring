package main

import (
	"github.com/MA-DOS/LowLevelMonitoring/client"
	"github.com/sirupsen/logrus"
)

const configFilePath = "config.yml"

func main() {
	// Parse the execution engine flag.
	engine := client.ParseExecutionEngineFlag()
	logrus.Infof("Using execution engine: %s", engine)

	// Load the configuration file.
	config, err := client.NewConfig(configFilePath)
	if err != nil {
		logrus.Error("Error reading config file: ", err)
		return
	}

	// Start the monitoring loop.
	client.ScheduleMonitoring(config, configFilePath, engine)
}
