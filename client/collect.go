package client

import (
	"sync"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// TODO: Helper functions are needed to split this
// Function to take in client configuration and queries to fetch monitoring targets in a thread.
func ShootQueries(c *Config, queries map[string][]string) (map[string][]model.Vector, []model.Vector, error) {
	resultsWithCategories := make(map[string][]model.Vector)
	resultsWithoutCategories := []model.Vector{}
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
				resultsWithCategories[metric] = append(resultsWithCategories[metric], fetcher)
				resultsWithoutCategories = append(resultsWithoutCategories, fetcher)
				mu.Unlock()
			}(metric, query)
		}
	}
	wg.Wait()
	logrus.Info("Amount of Threads: ", threadCounter)
	// logrus.Info("Results: ", resultsWithCategories)
	return resultsWithCategories, resultsWithoutCategories, nil
}

func StartMonitoring(c *Config, cfp string) (map[string][]model.Vector, []model.Vector, error) {
	resultMap, result, err := ShootQueries(c, ConsolidateQueries((ReadMonitoringConfiguration(cfp))))
	if err != nil {
		logrus.Error("Error shooting queries: ", err)
		return resultMap, result, err
	}
	return resultMap, result, nil
}
