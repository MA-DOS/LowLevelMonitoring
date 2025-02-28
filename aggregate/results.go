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
	Result    model.Vector                                  // Needed only for slurm_job_id metadata
	ResultMap map[string]map[string]map[string]model.Matrix // Holds the map according to the config structure
	// mu        sync.Mutex
}

func NewDataVectorWrapper(m map[string]map[string]map[string]model.Matrix) *DataVectorWrapper {
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
					for _, pair := range sample.Values {
						timestamp := pair.Timestamp.Time().Format("15:04:05.000")

						// logrus.Infof("Writing Sample - Timestamp: %s, ID: %s, Value: %f",
						// timestamp, sample.Metric["id"], float64(pair.Value))

						v.WriteToCSV(queryFolder, queryFile, timestamp, model.LabelSet(sample.Metric), float64(pair.Value))
					}
				}
			}
		}
	}
	return nil
}

func (v *DataVectorWrapper) WriteToCSV(folder string, outputFile *os.File, timestamp string, metricLabels model.LabelSet, value float64) error {
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
	values := ReadLabelValues(metricLabels, timestamp, value)

	// pattern := `^nxf-[A-Za-z0-9]+$`
	// re := regexp.MustCompile(pattern)
	// for _, taskName := range metricLabels {
	for range values {
		// fmt.Printf("Value: %f", value)
		// if re.MatchString(taskName) {
		// 	// fmt.Println("Found nextflow container")
		// 	FilterNextflowJobs(folder, taskName, values)
		// Also write all values into main file.
		if err := w.Write(values); err != nil {
			logrus.Error("Error writing to CSV")
			return err
		}
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
	logrus.Infof("Created output folder for finished Task: %s", containerName)
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
