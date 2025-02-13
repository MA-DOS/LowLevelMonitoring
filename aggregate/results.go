package aggregate

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

type DataVectorWrapper struct {
	Result    model.Vector                                  // Needed only for slurm_job_id metadata
	ResultMap map[string]map[string]map[string]model.Vector // Holds the map according to the config structure
	mu        sync.Mutex
}

func NewDataVectorWrapper(m map[string]map[string]map[string]model.Vector) *DataVectorWrapper {
	return &DataVectorWrapper{
		ResultMap: m,
	}
}

// This func is called on a MetaDataVectorWrapper object so it can access the fileds of the struct.
func (v *DataVectorWrapper) CreateDataOutput() error {
	err := CreateMonitoringOutput(v)
	if err != nil {
		logrus.Error("Error creating output structure: ", err)
	}
	return nil
}

// TODO: Go over the queries with separate go routines.
func CreateMonitoringOutput(v *DataVectorWrapper) error {
	err := os.Mkdir("results", 0755)
	if err != nil && !os.IsExist(err) {
		logrus.Error("Error creating results directory: ", err)
		return err
	}

	for target, dataSources := range v.ResultMap {
		targetFolder := fmt.Sprintf("results/%s", target)
		CreateOutputFolder(targetFolder)

		for dataSource, queryNames := range dataSources {
			sourceFolder := fmt.Sprintf("%s/%s", targetFolder, dataSource)
			CreateOutputFolder(sourceFolder)

			for queryName, samples := range queryNames {
				queryFolder := fmt.Sprintf("%s/%s", sourceFolder, queryName)
				CreateOutputFolder(queryFolder)

				queryFile := CreateFile(queryFolder, queryName+".csv")

				for _, sample := range samples {
					timestamp := sample.Timestamp.Time().Format("15:04:05.000")

					// logrus.Infof("Writing Sample - Timestamp: %s, ID: %s, Value: %f",
					// timestamp, sample.Metric["id"], float64(sample.Value))

					v.WriteToCSV(queryFile, timestamp, model.LabelSet(sample.Metric), float64(sample.Value))
				}
			}
		}
	}
	return nil
}

// func CreateMonitoringOutput(v *DataVectorWrapper) error {
// 	var wg sync.WaitGroup
// 	err := os.Mkdir("results", 0755)
// 	if err != nil && !os.IsExist(err) {
// 		logrus.Error("Error creating results directory: ", err)
// 		return err
// 	}

// 	for target, dataSources := range v.ResultMap {
// 		targetFolder := fmt.Sprintf("results/%s", target)
// 		CreateOutputFolder(targetFolder)

// 		for dataSource, queryNames := range dataSources {
// 			sourceFolder := fmt.Sprintf("%s/%s", targetFolder, dataSource)
// 			CreateOutputFolder(sourceFolder)

// 			for queryName, samples := range queryNames { // `samples` is of type `model.Vector` (which is []model.Sample)
// 				queryFolder := fmt.Sprintf("%s/%s", sourceFolder, queryName)
// 				CreateOutputFolder(queryFolder)

// 				queryFile := CreateFile(queryFolder, queryName+".csv")

// 				for _, sample := range samples {
// 					timestamp := sample.Timestamp.Time().Format("15:04:05.000")
// 					fmt.Printf("Timestamp: %s\n", timestamp)
// 					wg.Add(1)
// 					go func(sample model.Sample, timestamp string) {
// 						defer wg.Done()
// 						v.mu.Lock()
// 						defer v.mu.Unlock()
// 						v.WriteToCSV(queryFile, timestamp, model.LabelSet(sample.Metric), float64(sample.Value))
// 					}(*sample, timestamp)
// 				}
// 				// fmt.Printf("Timestamp Counter: %d\n", timestampCounter)
// 			}
// 		}
// 	}
// 	return nil
// }

// func CreateMonitoringOutput(v *DataVectorWrapper) error {
// 	// threadCounter := 0
// 	// var wg sync.WaitGroup

// 	err := os.Mkdir("results", 0755)
// 	if err != nil && !os.IsExist(err) {
// 		logrus.Error("Error creating results directory: ", err)
// 		return err
// 	}

// 	for target, dataSources := range v.ResultMap {

// 		targetFolder := fmt.Sprintf("results/%s", target)
// 		CreateOutputFolder(targetFolder)

// 		for dataSource, queryNames := range dataSources {
// 			sourceFolder := fmt.Sprintf("%s/%s", targetFolder, dataSource)
// 			CreateOutputFolder(sourceFolder)

// 			for queryName, vectors := range queryNames {
// 				queryFolder := fmt.Sprintf("%s/%s", sourceFolder, queryName)
// 				CreateOutputFolder(queryFolder)

// 				queryFile := CreateFile(queryFolder, queryName+".csv")

// 				for _, vector := range vectors {
// 					// fmt.Printf("Result Vector: %s", vector)

// 					if len(vector) == 0 {
// 						continue
// 					}

// 					for _, value := range vector {
// 						// fmt.Printf("Result Vector: %s", vector)

// 						timestamp := value.Timestamp.Time().Format("15:04:05")
// 						// fmt.Printf("Timestamp: %s\n", timestamp)
// 						// timestamp := value.Timestamp.Unix()
// 						// TODO: timestamp is incorrectly computed here.
// 						// timestamp := value.Timestamp.Time().Format("15:04:05")
// 						// wg.Add(1)
// 						// threadCounter++
// 						// go func(value model.Sample, timestamp string) {
// 						// 	defer wg.Done()
// 						// 	v.mu.Lock()
// 						fmt.Printf("Timestamp: %s\n", timestamp)
// 						// fmt.Printf("Query Samples: %s", value)
// 						// defer v.mu.Unlock()
// 						// timestamp := value.Timestamp.Time().Format("15:04:05")
// 						v.WriteToCSV(queryFile, timestamp, model.LabelSet(value.Metric), float64(value.Value))
// 						// }(*value, timestamp)
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }

func (v *DataVectorWrapper) WriteToCSV(outputFile *os.File, timestamp string, metricLabels model.LabelSet, value float64) error {
	w := csv.NewWriter(outputFile)
	w.Comma = ','
	defer w.Flush()

	fileInfo, err := outputFile.Stat()
	if err != nil {
		return err
	}
	if fileInfo.Size() == 0 {
		if err := w.Write(ReadHeaderFields(metricLabels)); err != nil {
			return err
		}
	}

	// Write data to the file.
	// TODO: Maybe not write the complete query output but sort by task name.
	values := ReadLabelValues(metricLabels, timestamp, value)
	if err := w.Write(values); err != nil {
		logrus.Error("Error writing to CSV")
		return err
	}
	return nil
}

// Helper to create folder.
func CreateOutputFolder(folderName string) error {
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		err := os.Mkdir(folderName, 0755)
		if err != nil {
			logrus.Error("Error creating folder: ", err)
		}
	}
	return nil
}

func CreateFile(path, fileName string) *os.File {
	fullPath := fmt.Sprintf("%s/%s", path, fileName)
	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error("Error opening file: ", err)
		return nil
	}
	return file
}

// Helper to get file name.
func GetFileName(targetFolder, fileIdentifier string) string {
	return fmt.Sprintf("%s/%s.csv", targetFolder, fileIdentifier)
}

func ReadHeaderFields(labelNames model.LabelSet) []string {
	// Check the current amount of fields in the header and check if it differs from the labelNames.
	// If it does, then we need to update the header.
	header := []string{"timestamp", "value"}
	keys := make([]string, 0, len(labelNames))
	for key := range labelNames {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	header = append(header, keys...)
	return header
}

func ReadLabelValues(labelValues model.LabelSet, timestamp string, value float64) []string {
	record := []string{
		// strconv.FormatInt(timestamp, 10),
		timestamp,
		fmt.Sprintf("%f", value),
	}

	keys := make([]string, 0, len(labelValues))
	for key := range labelValues {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	for _, key := range keys {
		record = append(record, string(labelValues[model.LabelName(key)]))
	}
	return record
}

func GetFileIdentifier(sample model.Sample) string {
	return sample.Value.String()
}

func GetTimeStamp(sample model.Sample) string {
	return sample.Timestamp.Time().Format("15:04:05")
}
