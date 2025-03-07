package client

import (
	"sync"
	"time"

	"github.com/barweiss/go-tuple"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// Function to take in client configuration and queries to fetch monitoring targets in a thread.
func FetchMonitoringSources(c *Config, cst, cdt time.Time, cn, cwdir, cid string, cpid int, queriesMap map[string]map[string][]tuple.T4[string, string, []string, string]) (map[string]map[string]map[string]model.Matrix, map[string][]string, error) {

	logrus.SetLevel(logrus.InfoLevel)
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
				go fetchQuery(c, target, dataSource, queryIdentifier, query, cst, cdt, cn, cwdir, cid, cpid, resultsWithCategories, &mu, &wg)
			}
		}
	}
	wg.Wait()
	return resultsWithCategories, queryMetaInfo, nil
}

func fetchQuery(c *Config, target, dataSource, queryIdentifier string, query tuple.T4[string, string, []string, string], cst, cdt time.Time, cn, cwdir, cid string, cpid int, mapTargetSourceName map[string]map[string]map[string]model.Matrix, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	client, err := NewFetchClient(c)
	if err != nil {
		logrus.Error("Error creating fetch client", err)
		return
	}

	// Insert the range for the query by event in the container engine.
	fetcher, err := FetchMonitoringTargets(client, queryIdentifier, query.V2, cst, cdt, cn, cwdir, cid, cpid)
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

func StartMonitoring(c *Config, cfp string, cst, cdt time.Time, cn, cwdir, cid string, cpid int) (map[string]map[string]map[string]model.Matrix, map[string][]string, error) {
	queriesMap := ConsolidateQueries(ReadMonitoringConfiguration(cfp)) // Ignore labels

	resultMap, QueryMetaInfo, err := FetchMonitoringSources(c, cst, cdt, cn, cwdir, cid, cpid, queriesMap)
	if err != nil {
		logrus.Error("Error fetching queries: ", err)
		return resultMap, QueryMetaInfo, err
	}
	return resultMap, QueryMetaInfo, err
}
