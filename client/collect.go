package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MA-DOS/LowLevelMonitoring/common"
	"github.com/barweiss/go-tuple"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// var maxRetries = 3
// var resultMatrix model.Matrix

// Using Prometheus API to fetch the monitoring targets.
func FetchMonitoringTargets(client api.Client, queryIdentifier, query string, entity common.WorkflowEntity) (model.Matrix, error) {
	v1api := v1.NewAPI(client)
	jobQuery := BuildQueryByLabelSelector(query, queryIdentifier, entity)
	logrus.Infof("Querying Prometheus with query: %s, Start Time: %s, End Time: %s, Step: %s", jobQuery, entity.GetStartTime().Format(time.RFC3339), entity.GetDieTime().Add(10*time.Second).Format(time.RFC3339), "500ms")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Perform range query
	// for attempt := 1; attempt <= maxRetries; attempt++ {
	result, warnings, err := v1api.QueryRange(ctx, jobQuery, v1.Range{
		Start: entity.GetStartTime(),                     // Subtract seconds to ensure I get the first sample.
		End:   entity.GetDieTime().Add(10 * time.Second), // Add seconds to ensure I get the last sample.
		Step:  500 * time.Millisecond,                    // Set the step to 500 milliseconds.
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

	// if len(resultMatrix) > 0 {
	// 	logrus.Infof("Prometheus query successful for query: %s, attempt: %d", jobQuery, attempt)
	// 	break
	// }

	logrus.Warnf("Prometheus query returned no results for query: %s", jobQuery)
	// if attempt < maxRetries {
	// 	time.Sleep(200 * time.Millisecond)
	// 	logrus.Infof("Retrying Prometheus query: %s, attempt: %d", jobQuery, attempt+1)
	// }
	// }

	if len(resultMatrix) == 0 {
		return nil, fmt.Errorf("no results found for query: %s", jobQuery)
	}
	return resultMatrix, nil
}

// Function to take in client configuration and queries to fetch monitoring targets in a thread.
func FetchMonitoringSources(c *Config, entity common.WorkflowEntity, queriesMap map[string]map[string][]tuple.T5[string, string, []string, string, string]) (map[string]map[string]map[string]model.Matrix, map[string][]string, map[string]map[string]string, error) {
	resultsWithCategories := make(map[string]map[string]map[string]model.Matrix)
	queryMetaInfo := make(map[string][]string)
	queryUnitInfo := make(map[string]map[string]string)

	logrus.Info("ENTERED")

	var mu sync.Mutex
	var wg sync.WaitGroup

	for target, dataSources := range queriesMap {
		for dataSource, queryList := range dataSources {
			if len(queryList) > 0 {

				queryMetaInfo[dataSource] = queryList[0].V3
				if _, ok := queryUnitInfo[dataSource]; !ok {
					queryUnitInfo[dataSource] = make(map[string]string)
				}
				for _, query := range queryList {
					queryUnitInfo[dataSource][query.V1] = query.V5
				}
			}
			for _, query := range queryList {
				wg.Add(1)

				queryIdentifier := queryList[0].V4
				logrus.Infof("Preparing to execute query for Target: %s, DataSource: %s, Query: %s", target, dataSource, query.V2)
				go fetchQuery(c, target, dataSource, queryIdentifier, query, entity, resultsWithCategories, &mu, &wg)
			}
		}
	}
	wg.Wait()
	return resultsWithCategories, queryMetaInfo, queryUnitInfo, nil
}

func fetchQuery(c *Config, target, dataSource, queryIdentifier string, query tuple.T5[string, string, []string, string, string], entity common.WorkflowEntity, mapTargetSourceName map[string]map[string]map[string]model.Matrix, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	client, err := NewFetchClient(c)
	if err != nil {
		logrus.Error("Error creating fetch client", err)
		return
	}

	// Insert the range for the query by event in the container engine.
	fetcher, err := FetchMonitoringTargets(client, queryIdentifier, query.V2, entity)
	if err != nil {
		logrus.Errorf("Error fetching monitoring targets for Target: %s, DataSource: %s, Query: %s - %v", target, dataSource, query.V2, err)
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
		logrus.Warnf("Query already exists for Target: %s, DataSource: %s, Query: %s", target, dataSource, query.V1)
	}
}

// Dynamically format the PromQL query based on the available identifiers.
func BuildQueryByLabelSelector(query, queryIdentifier string, entity common.WorkflowEntity) string {
	switch queryIdentifier {
	case "name":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, entity.GetName())
	case "path":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, entity.GetWorkDir())
	case "work_dir":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, entity.GetWorkDir())
	case "container_names":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, entity.GetName())
	case "container_name":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, entity.GetName())
	case "instance":
		// This is a temporary workaround as the snmp does not hold any container related identifiers.
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, "powermeter04.cit.tu-berlin.de")
	case "job":
		return fmt.Sprintf(`%s{%s="%s"}`, query, queryIdentifier, "ipmi_exporter")
	}
	return query
}
