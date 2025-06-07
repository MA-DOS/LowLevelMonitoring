package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MA-DOS/LowLevelMonitoring/watcher"
	"github.com/barweiss/go-tuple"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// Using Prometheus API to fetch the monitoring targets.
func FetchMonitoringTargets(client api.Client, queryIdentifier, query string, workflowContainer watcher.NextflowContainer) (model.Matrix, error) {
	v1api := v1.NewAPI(client)
	jobQuery := BuildQueryByLabelSelector(query, queryIdentifier, workflowContainer)
	logrus.Info("Querying Prometheus: ", jobQuery)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Perform range query
	result, warnings, err := v1api.QueryRange(ctx, jobQuery, v1.Range{
		Start: workflowContainer.StartTime,
		End:   workflowContainer.DieTime.Add(5 * time.Second), // Add seconds to ensure I get the last sample.
		Step:  500 * time.Millisecond,
	})

	if err != nil {
		return nil, fmt.Errorf("error querying Prometheus: %w", err)
	}
	if len(warnings) > 0 {
		logrus.Warnf("Prometheus warnings: %v", warnings)
	}

	resultMatrix, ok := result.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("failed to cast Prometheus response to Matrix")
	}
	if len(resultMatrix) == 0 {
		logrus.Warnf("Prometheus query returned no results for query: %s", jobQuery)
	}
	if !ok {
		return nil, fmt.Errorf("failed to cast Prometheus response to Matrix")
	}

	return resultMatrix, nil
}

// Function to take in client configuration and queries to fetch monitoring targets in a thread.
func FetchMonitoringSources(c *Config, workflowContainer watcher.NextflowContainer, queriesMap map[string]map[string][]tuple.T4[string, string, []string, string]) (map[string]map[string]map[string]model.Matrix, map[string][]string, error) {
	resultsWithCategories := make(map[string]map[string]map[string]model.Matrix)
	queryMetaInfo := make(map[string][]string)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for target, dataSources := range queriesMap {
		for dataSource, queryList := range dataSources {
			if len(queryList) > 0 {

				queryMetaInfo[dataSource] = queryList[0].V3
			}
			for _, query := range queryList {
				wg.Add(1)

				queryIdentifier := queryList[0].V4
				go fetchQuery(c, target, dataSource, queryIdentifier, query, workflowContainer, resultsWithCategories, &mu, &wg)
			}
		}
	}
	wg.Wait()
	return resultsWithCategories, queryMetaInfo, nil
}

func fetchQuery(c *Config, target, dataSource, queryIdentifier string, query tuple.T4[string, string, []string, string], workflowContainer watcher.NextflowContainer, mapTargetSourceName map[string]map[string]map[string]model.Matrix, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	client, err := NewFetchClient(c)
	if err != nil {
		logrus.Error("Error creating fetch client", err)
		return
	}

	// Insert the range for the query by event in the container engine.
	fetcher, err := FetchMonitoringTargets(client, queryIdentifier, query.V2, workflowContainer)
	if err != nil {
		logrus.Error("Error fetching monitoring targets", err)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := mapTargetSourceName[target]; !exists {
		mapTargetSourceName[target] = make(map[string]map[string]model.Matrix)
	}
	if _, exists := mapTargetSourceName[target][dataSource]; !exists {
		mapTargetSourceName[target][dataSource] = make(map[string]model.Matrix)
	}
	if _, exists := mapTargetSourceName[target][dataSource][query.V1]; !exists {
		mapTargetSourceName[target][dataSource][query.V1] = fetcher
	} else {
		logrus.Warn("Query already exists for target: ", target, " dataSource: ", dataSource, " query: ", query.V1)
	}
}

// Dynamically format the PromQL query based on the available identifiers.
func BuildQueryByLabelSelector(query, queryIdentifier string, workflowContainer watcher.NextflowContainer) string {
	switch queryIdentifier {
	case "name":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, workflowContainer.Name)
	case "path":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, workflowContainer.ContainerID)
	case "work_dir":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, workflowContainer.WorkDir)
	case "groupname":
		return fmt.Sprintf(`%s{%s="%v"}`, query, queryIdentifier, workflowContainer.PID)
	case "container_names":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, workflowContainer.Name)
	case "container_name":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, workflowContainer.Name)
	}
	return query
}
