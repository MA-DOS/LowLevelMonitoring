package client

import (
	"sync"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// Defines function type for Prometheus Client.
// Used to call FetchMonitoringTargets with the client configuration.
type PrometheusClient func(c *Config) (api.Client, error)

// Defines function type to consolidate init of client and queries.
// Used to call ShootQuery with the client configuration.
type PrometheusRequest func(client api.Client, query string)

// Function type for the queries and the actual request.
// Used so that in main only ScheduleMonitoring is called.
type MonitorQuery func(queries map[string][]string)

// Function to take in client configuration and queries to fetch monitoring targets in a thread.
func ShootQueries(c *Config, queries map[string][]string) (map[string][]model.Vector, error) {
	results := make(map[string][]model.Vector)
	var mu sync.Mutex
	var wg sync.WaitGroup
	// Debugging just for fun
	threadCounter := 0

	for metric, querySlices := range queries {
		for _, query := range querySlices {
			wg.Add(1)
			threadCounter++
			go func(metric, query string) {
				defer wg.Done()
				client, err := NewFetchClient(c)
				if err != nil {
					logrus.Error("Error creating fetch client", err)
					return
				}
				fetcher, err := FetchMonitoringTargets(client, query)
				if err != nil {
					logrus.Error("Error fetching monitoring targets", err)
					return
				}
				mu.Lock()
				results[metric] = append(results[metric], fetcher)
				mu.Unlock()
			}(metric, query)
		}
	}
	wg.Wait()
	logrus.Info("Amount of Threads: ", threadCounter)
	logrus.Info("Results: ", results)
	return results, nil
}

func ScheduleMonitoring(c *Config, cfp string) (map[string][]model.Vector, error) {
	result, err := ShootQueries(c, ConsolidateQueries((ReadMonitoringConfiguration(cfp))))
	if err != nil {
		logrus.Error("Error shooting queries: ", err)
		return result, err
	}
	return result, nil
}
