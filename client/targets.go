package client

import (
	"os"
	"regexp"

	"github.com/barweiss/go-tuple"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Returns a map of monitoring target to source and the respective queries per source
func ConsolidateQueries(mt map[string]interface{}) map[string]map[string][]tuple.T2[string, string] {
	sourceToQueryMap := BuildMetaDataQueries(mt)
	// cpuQueries := BuildCPUQueries(mt)
	// fmt.Println("CPU Queries: ", cpuQueries)
	// memoryQueries := BuildMemoryQueries(mt)
	// fmt.Println("Memory Queries: ", memoryQueries)
	// diskQueries := BuildDiskQueries(mt)
	// fmt.Println("Disk Queries: ", diskQueries)
	// networkQueries := BuildNetworkQueries(mt)
	// fmt.Println("Network Queries: ", networkQueries)
	// energyQueries := BuildEnergyQueries(mt)
	// fmt.Println("Energy Queries: ", energyQueries)

	return map[string]map[string][]tuple.T2[string, string]{
		"task_metadata": sourceToQueryMap,
	}
}

// Parse the config file for the metrics and according queries.
func ReadMonitoringConfiguration(cfp string) map[string]interface{} {
	// Read in the config yml.
	data, err := os.ReadFile(cfp)
	if err != nil {
		logrus.Error("Error with reading the config file, ", err)
	}
	// Save the output of the unmarshalling in a map.
	metricsMap := make(map[string]interface{})
	err = yaml.Unmarshal(data, &metricsMap)
	if err != nil {
		logrus.Error("Error unmarshalling the config file")
	}
	return metricsMap
}

func EscapeQuery(query string) string {
	re := regexp.MustCompile(`\\`)
	return re.ReplaceAllString(query, "")
}

func processSource(source interface{}, queriesPerDataSource map[string][]tuple.T2[string, string]) {
	sourceMap, ok := source.(map[string]interface{})
	if !ok {
		logrus.Warn("Source not defined for MetaData metrics")
		return
	}

	sourceName, ok := sourceMap["source"].(string)
	if !ok {
		logrus.Warn("Source name not defined for MetaData metrics")
		return
	}

	queriesPerDataSource[sourceName] = extractQueries(sourceMap)
}

func extractQueries(sourceMap map[string]interface{}) []tuple.T2[string, string] {
	metrics, ok := sourceMap["metrics"].([]interface{})
	if !ok || metrics == nil {
		logrus.Warn("Metrics not defined for MetaData metrics")
		return nil
	}

	var queries []tuple.T2[string, string]
	for _, metric := range metrics {
		if metricMap, ok := metric.(map[string]interface{}); ok {
			name, nameOk := metricMap["name"].(string)
			query, queryOk := metricMap["query"].(string)
			if nameOk && queryOk {
				queries = append(queries, tuple.New2(name, query))
			}
		}
	}
	return queries
}

func BuildMetaDataQueries(mt map[string]interface{}) map[string][]tuple.T2[string, string] {
	queriesPerDataSource := make(map[string][]tuple.T2[string, string])

	monitoringTargets, err := mt["monitoring_targets"].(map[string]interface{})
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}

	taskMetadata, ok := monitoringTargets["task_metadata"].(map[string]interface{})
	if !ok || !isEnabled(taskMetadata) {
		logrus.Warn("MetaData metrics not enabled")
		return queriesPerDataSource
	}

	metaDataSources, ok := taskMetadata["data_sources"].([]interface{})
	if !ok || metaDataSources == nil {
		logrus.Warn("Data sources not defined for MetaData metrics")
		return queriesPerDataSource
	}

	for _, source := range metaDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}

func isEnabled(target map[string]interface{}) bool {
	enabled, ok := target["enabled"].(bool)
	return ok && enabled
}

func BuildCPUQueries(mt map[string]interface{}) []string {
	monitoringTargets := mt["monitoring_targets"].(map[string]interface{})
	cpuMetrics := monitoringTargets["cpu"].(map[string]interface{})
	cpuMetricsEnabled := cpuMetrics["enabled"].(bool)
	var cpuQueries []string
	if cpuMetricsEnabled {
		if metrics, ok := cpuMetrics["metrics"].([]interface{}); ok && metrics != nil {
			for _, metric := range metrics {
				cpuMetricValues := metric.(map[string]interface{})
				if query, ok := cpuMetricValues["query"].(string); ok && query != "" {
					cleanQuery := EscapeQuery(query)
					cpuQueries = append(cpuQueries, cleanQuery)
				} else {
					logrus.Warn("Query not defined for CPU metrics")
				}
			}
		} else {
			logrus.Warn("Metrics need to be defined for every monitoring target")
		}
	}
	return cpuQueries
}

func BuildMemoryQueries(mt map[string]interface{}) []string {
	monitoringTargets := mt["monitoring_targets"].(map[string]interface{})
	memoryMetrics := monitoringTargets["memory"].(map[string]interface{})
	memoryMetricsEnabled := memoryMetrics["enabled"].(bool)
	var memoryQueries []string
	if memoryMetricsEnabled {
		if metrics, ok := memoryMetrics["metrics"].([]interface{}); ok && metrics != nil {
			for _, metric := range metrics {
				memoryMetricValues := metric.(map[string]interface{})
				if query, ok := memoryMetricValues["query"].(string); ok && query != "" {
					cleanQuery := EscapeQuery(query)
					memoryQueries = append(memoryQueries, cleanQuery)
				} else {
					logrus.Warn("Query not defined for Memory metrics")
				}
			}
		} else {
			logrus.Warn("Metrics need to be defined for every monitoring target")
		}
	}
	return memoryQueries
}

func BuildDiskQueries(mt map[string]interface{}) []string {
	monitoringTargets := mt["monitoring_targets"].(map[string]interface{})
	diskMetrics := monitoringTargets["disk"].(map[string]interface{})
	diskMetricsEnabled := diskMetrics["enabled"].(bool)
	var diskQueries []string
	if diskMetricsEnabled {
		if metrics, ok := diskMetrics["metrics"].([]interface{}); ok && metrics != nil {
			for _, metric := range metrics {
				diskMetricValues := metric.(map[string]interface{})
				if query, ok := diskMetricValues["query"].(string); ok && query != "" {
					cleanQuery := EscapeQuery(query)
					diskQueries = append(diskQueries, cleanQuery)
				} else {
					logrus.Warn("Query not defined for Disk metrics")
				}
			}
		} else {
			logrus.Warn("Metrics need to be defined for every monitoring target")
		}
	}
	return diskQueries
}

func BuildNetworkQueries(mt map[string]interface{}) []string {
	monitoringTargets := mt["monitoring_targets"].(map[string]interface{})
	networkMetrics := monitoringTargets["network"].(map[string]interface{})
	networkMetricsEnabled := networkMetrics["enabled"].(bool)
	var networkQueries []string
	if networkMetricsEnabled {
		if metrics, ok := networkMetrics["metrics"].([]interface{}); ok && metrics != nil {
			for _, metric := range metrics {
				networkMetricValues := metric.(map[string]interface{})
				if query, ok := networkMetricValues["query"].(string); ok && query != "" {
					cleanQuery := EscapeQuery(query)
					networkQueries = append(networkQueries, cleanQuery)
				} else {
					logrus.Warn("Query not defined for Network metrics")
				}
			}
		} else {
			logrus.Warn("Metrics need to be defined for every monitoring target")
		}
	}
	return networkQueries
}

func BuildEnergyQueries(mt map[string]interface{}) []string {
	monitoringTargets := mt["monitoring_targets"].(map[string]interface{})
	energyMetrics := monitoringTargets["energy"].(map[string]interface{})
	energyMetricsEnabled := energyMetrics["enabled"].(bool)
	var energyQueries []string
	if energyMetricsEnabled {
		if metrics, ok := energyMetrics["metrics"].([]interface{}); ok && metrics != nil {
			for _, metric := range metrics {
				energyMetricValues := metric.(map[string]interface{})
				if query, ok := energyMetricValues["query"].(string); ok && query != "" {
					cleanQuery := EscapeQuery(query)
					energyQueries = append(energyQueries, cleanQuery)
				} else {
					logrus.Warn("Query not defined for Energy metrics")
				}
			}
		} else {
			logrus.Warn("Metrics need to be defined for every monitoring target")
		}
	}
	return energyQueries
}
