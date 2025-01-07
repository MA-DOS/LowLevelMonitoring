package client

import (
	"fmt"
	"os"
	"regexp"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func ConsolidateQueries(mt map[string]interface{}) map[string][]string {
	// Debugging
	// fmt.Println("Target Aggregation active: ", mt)

	metaDataQueries := BuildMetaDataQueries(mt)
	fmt.Println("MetaData Queries: ", metaDataQueries)

	cpuQueries := BuildCPUQueries(mt)
	fmt.Println("CPU Queries: ", cpuQueries)

	memoryQueries := BuildMemoryQueries(mt)
	fmt.Println("Memory Queries: ", memoryQueries)

	diskQueries := BuildDiskQueries(mt)
	fmt.Println("Disk Queries: ", diskQueries)

	networkQueries := BuildNetworkQueries(mt)
	fmt.Println("Network Queries: ", networkQueries)

	energyQueries := BuildEnergyQueries(mt)
	fmt.Println("Energy Queries: ", energyQueries)

	return map[string][]string{
		"metaDataQueries": metaDataQueries,
		"cpuQueries":      cpuQueries,
		"memoryQueries":   memoryQueries,
		"diskQueries":     diskQueries,
		"networkQueries":  networkQueries,
		"energyQueries":   energyQueries,
	}
}

// Parse the config file for the metrics and according queries.
func ReadMonitoringConfiguration(cfp string) map[string]interface{} {
	// Read in the config yml.
	data, err := os.ReadFile(cfp)
	if err != nil {
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

func BuildMetaDataQueries(mt map[string]interface{}) []string {
	monitoringTargets := mt["monitoring_targets"].(map[string]interface{})
	taskMetadata := monitoringTargets["task_metadata"].(map[string]interface{})
	metaDataEnabled := taskMetadata["enabled"].(bool)
	var metaDataQueries []string
	if metaDataEnabled {
		if metrics, ok := taskMetadata["metrics"].([]interface{}); ok && metrics != nil {
			for _, metric := range metrics {
				metaDataMetricValues := metric.(map[string]interface{})
				if query, ok := metaDataMetricValues["query"].(string); ok && query != "" {

					cleanQuery := EscapeQuery(query)
					metaDataQueries = append(metaDataQueries, cleanQuery)
				} else {
					logrus.Warn("Query not defined for MetaData metrics")
				}
			}

		} else {
			logrus.Warn("Metrics need to be defined for every monitoring target")
		}
	}
	return metaDataQueries
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
					logrus.Warn("Query not defined for CPU metrics")
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
