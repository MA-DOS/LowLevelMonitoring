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
		return
	}
	// Get the monitoring interval from the configuration file.
	monitoringInterval := config.ServerConfigurations.Prometheus.TargetServer.FetchInterval

	// Start the monitoring loop.
	ScheduleMonitoring(config, monitoringInterval)
}

// TODO: Refactor to read path out of config, somehow did not work...
func ScheduleMonitoring(config *client.Config, interval int) {
	for {
		// Run the Monitor
		resultMap, results, err := client.StartMonitoring(config, configFilePath)
		if err != nil {
			logrus.Error("Error starting monitoring: ", err)
			// panic(err)
		}

		// Data Structure for meta data results
		for _, result := range results {
			metaDataWrapper := aggregate.NewMetaDataVectorWrapper(result, resultMap)
			fmt.Println("MetaDataWrapper: ", metaDataWrapper)

			err = metaDataWrapper.CreateMetaDataOutput()
			if err != nil {
				logrus.Error("Error creating output: ", err)
			}
		}
		interval := config.ServerConfigurations.Prometheus.TargetServer.FetchInterval
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
