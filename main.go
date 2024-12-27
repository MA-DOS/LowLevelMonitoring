package main

import (
	"github.com/sirupsen/logrus"

	"github.com/MA-DOS/LowLevelMonitoring/client"
)

const configFilePath = "../config/config.yaml"

func main() {
	// Load the configuration from the file for the client.
	config, err := client.LoadConfig(configFilePath)
	if err != nil {
		logrus.Fatalf("failed to load config: %v", err)
	}
	logrus.Printf("Server address: %s", config.Server.Address)

	// Create a new client based on the configuration.
	c, err := client.NewClient(*config)
	if err != nil {
		logrus.Fatalf("failed to create client: %v", err)
	}
	logrus.Info("Client created: %v", c)

	// Lauch the clients seperately per metric and let them collect the data and write them to csv.
}
