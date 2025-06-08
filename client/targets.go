package client

import (
	"os"
	"regexp"

	"github.com/barweiss/go-tuple"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Returns a map of monitoring target to source and the respective queries per source
func ConsolidateQueries(mt map[string]any) map[string]map[string][]tuple.T5[string, string, []string, string, string] {
	sourceToMetaQueryMap := BuildMetaDataQueries(mt)
	sourceToCpuQueryMap := BuildCPUQueries(mt)
	sourceToMemoryMap := BuildMemoryQueries(mt)
	sourceToDiskMap := BuildDiskQueries(mt)
	sourceToNetworkMap := BuildNetworkQueries(mt)
	sourceToEnergyMap := BuildEnergyQueries(mt)

	return map[string]map[string][]tuple.T5[string, string, []string, string, string]{
		"task_metadata":     sourceToMetaQueryMap,
		"task_cpu_data":     sourceToCpuQueryMap,
		"task_memory_data":  sourceToMemoryMap,
		"task_disk_data":    sourceToDiskMap,
		"task_network_data": sourceToNetworkMap,
		"task_energy_data":  sourceToEnergyMap,
	}
}

// Parse the config file for the metrics and according queries.
func ReadMonitoringConfiguration(cfp string) map[string]any {
	// Read in the config yml.
	data, err := os.ReadFile(cfp)
	if err != nil {
		logrus.Error("Error with reading the config file, ", err)
	}
	// Save the output of the unmarshalling in a map.
	metricsMap := make(map[string]any)
	err = yaml.Unmarshal(data, &metricsMap)
	if err != nil {
		logrus.Error("Error unmarshalling the config file")
	}
	return metricsMap
}

func extractLabelSet(sourceMap map[string]any) []string {
	labels, ok := sourceMap["labels"].([]any)
	if !ok || labels == nil {
		logrus.Warn("Labels need to be defined for a cleaned CSV extraction...!")
		return nil
	}
	var labelSet []string
	for _, label := range labels {
		if labelStr, ok := label.(string); ok {
			labelSet = append(labelSet, labelStr)
		}
	}
	return labelSet
}

func extractQueryIdentifier(sourceMap map[string]any) string {
	identifier, ok := sourceMap["identifier"].(string)
	if !ok {
		logrus.Warn("Identifier not defined...")
		return ""
	}
	return identifier
}

func EscapeQuery(query string) string {
	re := regexp.MustCompile(`\\`)
	return re.ReplaceAllString(query, "")
}

func processSource(source any, queriesPerDataSource map[string][]tuple.T5[string, string, []string, string, string]) {
	sourceMap := source.(map[string]any)

	sourceName, ok := sourceMap["source"].(string)
	if !ok {
		logrus.Warn("Source name not defined...")
		return
	}

	queriesPerDataSource[sourceName] = extractQueryData(sourceMap)
}

func extractQueryData(sourceMap map[string]any) []tuple.T5[string, string, []string, string, string] {
	metrics, ok := sourceMap["metrics"].([]any)
	if !ok || metrics == nil {
		logrus.Warn("Metrics need to be defined")
		return nil
	}
	var queries []tuple.T5[string, string, []string, string, string]
	for _, metric := range metrics {
		if metricMap, ok := metric.(map[string]any); ok {
			labelSet := extractLabelSet(sourceMap)
			name, nameOk := metricMap["name"].(string)
			query, queryOk := metricMap["query"].(string)
			unit, unitOk := metricMap["unit"].(string)
			if nameOk && queryOk && unitOk {
				queries = append(queries, tuple.New5(name, query, labelSet, extractQueryIdentifier(sourceMap), unit))
			}
		}
	}
	return queries
}

func BuildMetaDataQueries(mt map[string]any) map[string][]tuple.T5[string, string, []string, string, string] {
	queriesPerDataSource := make(map[string][]tuple.T5[string, string, []string, string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]any)
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}

	taskMetadata, ok := monitoringTargets["task_metadata"].(map[string]any)
	if !ok || !isEnabled(taskMetadata) {
		// logrus.Warn("MetaData metrics not enabled")
		return queriesPerDataSource
	}

	metaDataSources, ok := taskMetadata["data_sources"].([]any)
	if !ok || metaDataSources == nil {
		logrus.Warn("Data sources not defined for MetaData metrics")
		return queriesPerDataSource
	}

	for _, source := range metaDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}

func isEnabled(target map[string]any) bool {
	enabled, ok := target["enabled"].(bool)
	return ok && enabled
}

func BuildCPUQueries(mt map[string]any) map[string][]tuple.T5[string, string, []string, string, string] {
	queriesPerDataSource := make(map[string][]tuple.T5[string, string, []string, string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]any)
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}
	taskCpuData, ok := monitoringTargets["cpu"].(map[string]any)
	if !ok || !isEnabled(taskCpuData) {
		// logrus.Warn("CPU metrics not enabled")
		return queriesPerDataSource
	}

	cpuDataSources, ok := taskCpuData["data_sources"].([]any)
	if !ok || cpuDataSources == nil {
		logrus.Warn("Data sources not defined for CPU metrics")
		return queriesPerDataSource
	}

	for _, source := range cpuDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}

func BuildMemoryQueries(mt map[string]any) map[string][]tuple.T5[string, string, []string, string, string] {
	queriesPerDataSource := make(map[string][]tuple.T5[string, string, []string, string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]any)
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}
	taskMemoryData, ok := monitoringTargets["memory"].(map[string]any)
	if !ok || !isEnabled(taskMemoryData) {
		// logrus.Warn("Memory metrics not enabled")
		return nil
	}

	memoryDataSources, ok := taskMemoryData["data_sources"].([]any)
	if !ok || memoryDataSources == nil {
		logrus.Warn("Data sources not defined for Memory metrics")
		return nil
	}

	for _, source := range memoryDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}

func BuildDiskQueries(mt map[string]any) map[string][]tuple.T5[string, string, []string, string, string] {
	queriesPerDataSource := make(map[string][]tuple.T5[string, string, []string, string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]any)
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}
	taskDiskData, ok := monitoringTargets["disk"].(map[string]any)
	if !ok || !isEnabled(taskDiskData) {
		// logrus.Warn("Disk metrics not enabled")
		return queriesPerDataSource
	}

	diskDataSources, ok := taskDiskData["data_sources"].([]any)
	if !ok || diskDataSources == nil {
		logrus.Warn("Data sources not defined for Disk metrics")
		return queriesPerDataSource
	}

	for _, source := range diskDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}

func BuildNetworkQueries(mt map[string]any) map[string][]tuple.T5[string, string, []string, string, string] {
	queriesPerDataSource := make(map[string][]tuple.T5[string, string, []string, string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]any)
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}
	taskNetworkData, ok := monitoringTargets["network"].(map[string]any)
	if !ok || !isEnabled(taskNetworkData) {
		// logrus.Warn("Network metrics not enabled")
		return queriesPerDataSource
	}

	networkDataSources, ok := taskNetworkData["data_sources"].([]any)
	if !ok || networkDataSources == nil {
		logrus.Warn("Data sources not defined for Network metrics")
		return queriesPerDataSource
	}

	for _, source := range networkDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}

func BuildEnergyQueries(mt map[string]any) map[string][]tuple.T5[string, string, []string, string, string] {
	queriesPerDataSource := make(map[string][]tuple.T5[string, string, []string, string, string])
	monitoringTargets, err := mt["monitoring_targets"].(map[string]any)
	if !err {
		logrus.Error("Monitoring targets not defined, please check the config file")
	}
	taskEnergyData, ok := monitoringTargets["energy"].(map[string]any)
	if !ok || !isEnabled(taskEnergyData) {
		// logrus.Warn("Energy metrics not enabled")
		return queriesPerDataSource
	}

	energyDataSources, ok := taskEnergyData["data_sources"].([]any)
	if !ok || energyDataSources == nil {
		logrus.Warn("Data sources not defined for Energy metrics")
		return queriesPerDataSource
	}

	for _, source := range energyDataSources {
		processSource(source, queriesPerDataSource)
	}
	return queriesPerDataSource
}
