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
	sourceToMetaQueryMap := BuildMetaDataQueries(mt)
	sourceToCpuQueryMap := BuildCPUQueries(mt)
	sourceToMemoryMap := BuildMemoryQueries(mt)
	sourceToDiskMap := BuildDiskQueries(mt)
	sourceToNetworkMap := BuildNetworkQueries(mt)
	sourceToEnergyMap := BuildEnergyQueries(mt)

	return map[string]map[string][]tuple.T2[string, string]{
		"task_metadata":     sourceToMetaQueryMap,
		"task_cpu_data":     sourceToCpuQueryMap,
		"task_memory_data":  sourceToMemoryMap,
		"task_disk_data":    sourceToDiskMap,
		"task_network_data": sourceToNetworkMap,
		"task_energy_data":  sourceToEnergyMap,
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
		logrus.Warn("Metrics need to be defined")
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

func BuildCPUQueries(mt map[string]interface{}) map[string][]tuple.T2[string, string] {
	queriesPerDataSource := make(map[string][]tuple.T2[string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]interface{})
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}
	taskCpuData, ok := monitoringTargets["cpu"].(map[string]interface{})
	if !ok || !isEnabled(taskCpuData) {
		logrus.Warn("CPU metrics not enabled")
		return queriesPerDataSource
	}

	cpuDataSources, ok := taskCpuData["data_sources"].([]interface{})
	if !ok || cpuDataSources == nil {
		logrus.Warn("Data sources not defined for CPU metrics")
		return queriesPerDataSource
	}

	for _, source := range cpuDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}

func BuildMemoryQueries(mt map[string]interface{}) map[string][]tuple.T2[string, string] {
	queriesPerDataSource := make(map[string][]tuple.T2[string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]interface{})
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}
	taskMemoryData, ok := monitoringTargets["memory"].(map[string]interface{})
	if !ok || !isEnabled(taskMemoryData) {
		logrus.Warn("Memory metrics not enabled")
		return nil
	}

	memoryDataSources, ok := taskMemoryData["data_sources"].([]interface{})
	if !ok || memoryDataSources == nil {
		logrus.Warn("Data sources not defined for Memory metrics")
		return nil
	}

	for _, source := range memoryDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}

func BuildDiskQueries(mt map[string]interface{}) map[string][]tuple.T2[string, string] {
	queriesPerDataSource := make(map[string][]tuple.T2[string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]interface{})
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}
	taskDiskData, ok := monitoringTargets["disk"].(map[string]interface{})
	if !ok || !isEnabled(taskDiskData) {
		logrus.Warn("Disk metrics not enabled")
		return queriesPerDataSource
	}

	diskDataSources, ok := taskDiskData["data_sources"].([]interface{})
	if !ok || diskDataSources == nil {
		logrus.Warn("Data sources not defined for Disk metrics")
		return queriesPerDataSource
	}

	for _, source := range diskDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}

func BuildNetworkQueries(mt map[string]interface{}) map[string][]tuple.T2[string, string] {
	queriesPerDataSource := make(map[string][]tuple.T2[string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]interface{})
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}
	taskNetworkData, ok := monitoringTargets["network"].(map[string]interface{})
	if !ok || !isEnabled(taskNetworkData) {
		logrus.Warn("Network metrics not enabled")
		return queriesPerDataSource
	}

	networkDataSources, ok := taskNetworkData["data_sources"].([]interface{})
	if !ok || networkDataSources == nil {
		logrus.Warn("Data sources not defined for Network metrics")
		return queriesPerDataSource
	}

	for _, source := range networkDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}

func BuildEnergyQueries(mt map[string]interface{}) map[string][]tuple.T2[string, string] {
	queriesPerDataSource := make(map[string][]tuple.T2[string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]interface{})
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}
	taskEnergyData, ok := monitoringTargets["energy"].(map[string]interface{})
	if !ok || !isEnabled(taskEnergyData) {
		logrus.Warn("Energy metrics not enabled")
		return queriesPerDataSource
	}

	energyDataSources, ok := taskEnergyData["data_sources"].([]interface{})
	if !ok || energyDataSources == nil {
		logrus.Warn("Data sources not defined for Energy metrics")
		return queriesPerDataSource
	}

	for _, source := range energyDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}
