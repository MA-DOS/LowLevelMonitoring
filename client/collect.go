package client

import (
	"sync"
	"time"

	"github.com/barweiss/go-tuple"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// TODO: Helper functions are needed to split this
// Function to take in client configuration and queries to fetch monitoring targets in a thread.
func FetchMonitoringSources(c *Config, cst, cdt time.Time, cn string, queries map[string]map[string][]tuple.T2[string, string]) (map[string]map[string]map[string]model.Vector, error) {
	logrus.SetLevel(logrus.InfoLevel)
	resultsWithCategories := make(map[string]map[string]map[string]model.Vector)
	// resultsWithoutCategories := model.Vector{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	// Debugging just for fun
	threadCounter := 0

	for target, dataSources := range queries {
		for dataSource, querySlices := range dataSources {
			for _, query := range querySlices {
				wg.Add(1)
				threadCounter++
				go fetchQuery(c, target, dataSource, query, cst, cdt, cn, resultsWithCategories, &mu, &wg)
			}
		}
	}
	wg.Wait()
	// fmt.Println("resultsWithCategories: ", resultsWithCategories)
	return resultsWithCategories, nil
}

func fetchQuery(c *Config, target, dataSource string, query tuple.T2[string, string], cst, cdt time.Time, cn string, mapTargetSourceName map[string]map[string]map[string]model.Vector, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	client, err := NewFetchClient(c)
	if err != nil {
		logrus.Error("Error creating fetch client", err)
		return
	}

	// Insert the range for the query by event in the container engine.
	// queryString := strings.Replace(query.V2, "[DURATION]", duration, -1)
	fetcher, err := FetchMonitoringTargets(client, query.V2, cst, cdt, cn)
	if err != nil {
		logrus.Error("Error fetching monitoring targets", err)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := mapTargetSourceName[target]; !exists {
		mapTargetSourceName[target] = make(map[string]map[string]model.Vector)
	}
	if _, exists := mapTargetSourceName[target][dataSource]; !exists {
		mapTargetSourceName[target][dataSource] = make(map[string]model.Vector)
	}
	if _, exists := mapTargetSourceName[target][dataSource][query.V1]; !exists {
		mapTargetSourceName[target][dataSource][query.V1] = model.Vector{}
	}

	for _, sample := range fetcher {
		found := false
		for _, existingSample := range (mapTargetSourceName)[target][dataSource][query.V1] {
			if existingSample.Timestamp == sample.Timestamp && existingSample.Metric.Equal(sample.Metric) {
				found = true
				break
			}
		}
		if !found {
			(mapTargetSourceName)[target][dataSource][query.V1] = append(
				(mapTargetSourceName)[target][dataSource][query.V1], sample)
		}
	}
}

func StartMonitoring(c *Config, cfp string, cst, cdt time.Time, cn string) (map[string]map[string]map[string]model.Vector, error) {
	resultMap, err := FetchMonitoringSources(c, cst, cdt, cn, ConsolidateQueries((ReadMonitoringConfiguration(cfp))))
	if err != nil {
		logrus.Error("Error shooting queries: ", err)
		return resultMap, err
	}
	// fmt.Printf("resultMap: %v\n", resultMap)
	return resultMap, nil
}
