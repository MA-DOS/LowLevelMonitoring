package aggregate

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

type DataVectorWrapper struct {
	Result        model.Vector                                  // Needed only for slurm_job_id metadata
	ResultMap     map[string]map[string]map[string]model.Matrix // Holds the map according to the config structure
	QueryMetaInfo map[string][]string
	QueryUnits    map[string]map[string]string
	// mu        sync.Mutex
}

func NewDataVectorWrapper(m map[string]map[string]map[string]model.Matrix, qmi map[string][]string, qu map[string]map[string]string) *DataVectorWrapper {
	return &DataVectorWrapper{
		ResultMap:     m,
		QueryMetaInfo: qmi,
		QueryUnits:    qu,
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
				var unit string
				if v.QueryUnits != nil {
					if unitsForSource, ok := v.QueryUnits[dataSource]; ok {
						unit = unitsForSource[queryName]
						fmt.Printf("Unit for %s/%s: %s\n", dataSource, queryName, unit)
					}
				}

				for _, sample := range samples {
					for _, pair := range sample.Values {
						timestamp := pair.Timestamp.Time().Format("15:04:05.000")

						// logrus.Infof("Writing Sample - Timestamp: %s, ID: %s, Value: %f",
						// timestamp, sample.Metric["id"], float64(pair.Value))

						v.WriteToCSV(dataSource, v.QueryMetaInfo, queryFolder, queryFile, timestamp, model.LabelSet(sample.Metric), float64(pair.Value), unit)
						// logrus.Info("[QUERY Labels]: ", v.QueryMetaInfo)
					}
				}
			}
		}
	}
	return nil
}

func (v *DataVectorWrapper) WriteToCSV(dataSource string, QueryMetaInfo map[string][]string, folder string, outputFile *os.File, timestamp string, metricLabels model.LabelSet, value float64, unit string) error {
	w := csv.NewWriter(outputFile)
	w.Comma = ','
	defer w.Flush()

	fileInfo, err := outputFile.Stat()
	if err != nil {
		return err
	}
	// if fileInfo.Size() == 0 {
	// 	if err := w.Write(ReadHeaderFields(metricLabels)); err != nil {
	// 		return err
	// 	}
	// }
	if fileInfo.Size() == 0 {
		if err := w.Write(ReadHeaderFields(dataSource, QueryMetaInfo, unit)); err != nil {
			return err
		}
	}

	// Write data to the file.
	values := ReadLabelValues(dataSource, QueryMetaInfo, metricLabels, timestamp, value)

	// pattern := `^nxf-[A-Za-z0-9]+$`
	// re := regexp.MustCompile(pattern)
	// for _, taskName := range metricLabels {
	// fmt.Printf("Value: %f", value)
	// if re.MatchString(taskName) {
	// 	// fmt.Println("Found nextflow container")
	// 	FilterNextflowJobs(folder, taskName, values)
	// Also write all values into main file.
	if err := w.Write(values); err != nil {
		logrus.Error("Error writing to CSV")
		return err
	}
	return nil
}

// if err := w.Write(values); err != nil {
// 	logrus.Error("Error writing to CSV")
// 	return err
// }
// 	return nil
// }

func FilterNextflowJobs(queryFolder string, taskName string, values []string) {
	containerName := taskName

	taskFolder, err := CreateOutputFolder(fmt.Sprintf("%s/%s", queryFolder, containerName))
	// logrus.Infof("Created output folder for finished Task: %s", containerName)
	if err != nil {
		logrus.Error("Error creating task folder: ", err)
	}
	taskFile := CreateFile(taskFolder, containerName+".csv")
	if taskFile == nil {
		logrus.Error("Error creating task file")
		return
	}

	tw := csv.NewWriter(taskFile)
	tw.Comma = ','
	defer tw.Flush()
	if err := tw.Write(values); err != nil {
		logrus.Error("Error writing to CSV")

	}
	// fmt.Println("Values: ", values)
}

// Helper to create folder.
func CreateOutputFolder(folderName string) (path string, err error) {
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		err := os.MkdirAll(folderName, 0755)
		if err != nil {
			logrus.Error("Error creating folder: ", err)
			return "", err
		}
	}
	return folderName, nil
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

// func ReadHeaderFields(labelNames model.LabelSet) []string {
// 	// Check the current amount of fields in the header and check if it differs from the labelNames.
// 	// If it does, then we need to update the header.
// 	header := []string{"timestamp", "value"}
// 	keys := make([]string, 0, len(labelNames))
// 	for key := range labelNames {
// 		keys = append(keys, string(key))
// 	}
// 	sort.Strings(keys)
// 	header = append(header, keys...)
// 	return header
// }

// Don't read header fields from model.LabelSet but from the config file.
func ReadHeaderFields(dataSource string, QueryMetaInfo map[string][]string, unit string) []string {
	// Check the current amount of fields in the header and check if it differs from the labelNames.
	// If it does, then we need to update the header.
	valueWithUnit := fmt.Sprint("Value (", unit, ")")
	header := []string{"timestamp", valueWithUnit}
	if unit == "" {
		valueWithUnit = "Value"
	}
	if fields, ok := QueryMetaInfo[dataSource]; ok {
		header = append(header, fields...)
	}
	return header
	// fieldSet := make(map[string]struct{})

	// for _, fields := range QueryMetaInfo {
	// 	for _, field := range fields {
	// 		fieldSet[field] = struct{}{}
	// 	}
	// }
	// keys := make([]string, 0, len(fieldSet))
	// for field := range fieldSet {
	// 	keys = append(keys, field)
	// }
	// sort.Strings(keys)
	// header = append(header, keys...)
	// return header
}

func ReadLabelValues(dataSource string, QueryMetaInfo map[string][]string, labelValues model.LabelSet, timestamp string, value float64) []string {
	record := []string{
		// strconv.FormatInt(timestamp, 10),
		timestamp,
		fmt.Sprintf("%f", value),
	}

	labels, ok := QueryMetaInfo[dataSource]
	if !ok {
		logrus.Warn("Labels not found for source: ", dataSource)
		return nil
	}

	keys := make([]string, 0, len(labels))
	keys = append(keys, labels...)
	// logrus.Infof("QueryMetaInfo: %v", QueryMetaInfo)
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
