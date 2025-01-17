package client

import (
	"sync"

	"github.com/barweiss/go-tuple"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// TODO: Helper functions are needed to split this
// Function to take in client configuration and queries to fetch monitoring targets in a thread.
func FetchMonitoringSources(c *Config, queries map[string]map[string][]tuple.T2[string, string]) (map[string]map[string]map[string][]model.Vector, []model.Vector, error) {
	logrus.SetLevel(logrus.InfoLevel)
	resultsWithCategories := make(map[string]map[string]map[string][]model.Vector)
	resultsWithoutCategories := []model.Vector{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	// Debugging just for fun
	threadCounter := 0

	for target, dataSources := range queries {
		for dataSource, querySlices := range dataSources {
			for _, query := range querySlices {
				wg.Add(1)
				threadCounter++
				go fetchQuery(c, target, dataSource, query, &resultsWithCategories, &resultsWithoutCategories, &mu, &wg)
			}
		}
	}
	wg.Wait()
	logrus.Info("Threads fetching metrics: ", threadCounter)
	return resultsWithCategories, resultsWithoutCategories, nil
}

func fetchQuery(c *Config, target, dataSource string, query tuple.T2[string, string], mapTargetSourceName *map[string]map[string]map[string][]model.Vector, onlyVectorMap *[]model.Vector, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	client, err := NewFetchClient(c)
	if err != nil {
		logrus.Error("Error creating fetch client", err)
		return
	}
	fetcher, err := FetchMonitoringTargets(client, query.V2)
	if err != nil {
		logrus.Error("Error fetching monitoring targets", err)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if _, exists := (*mapTargetSourceName)[target]; !exists {
		(*mapTargetSourceName)[target] = make(map[string]map[string][]model.Vector)
	}
	if _, exists := (*mapTargetSourceName)[target][dataSource]; !exists {
		(*mapTargetSourceName)[target][dataSource] = make(map[string][]model.Vector)
	}
	(*mapTargetSourceName)[target][dataSource][query.V1] = append((*mapTargetSourceName)[target][dataSource][query.V1], fetcher)
	*onlyVectorMap = append(*onlyVectorMap, fetcher)
}

func StartMonitoring(c *Config, cfp string) (map[string]map[string]map[string][]model.Vector, []model.Vector, error) {
	resultMap, result, err := FetchMonitoringSources(c, ConsolidateQueries((ReadMonitoringConfiguration(cfp))))
	if err != nil {
		logrus.Error("Error shooting queries: ", err)
		return resultMap, result, err
	}
	return resultMap, result, nil
}
