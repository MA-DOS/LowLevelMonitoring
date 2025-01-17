package aggregate

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

type MetaDataVectorWrapper struct {
	Result    model.Vector                                    // Needed only for slurm_job_id metadata
	ResultMap map[string]map[string]map[string][]model.Vector // Holds the map according to the config structure
}

func NewMetaDataVectorWrapper(r model.Vector, m map[string]map[string]map[string][]model.Vector) *MetaDataVectorWrapper {
	return &MetaDataVectorWrapper{
		Result:    r,
		ResultMap: m,
	}
}

// This func is called on a MetaDataVectorWrapper object so it can access the fileds of the struct.
func (v *MetaDataVectorWrapper) CreateMetaDataOutput() error {
	err := CreateMonitoringOutput(v)
	if err != nil {
		logrus.Error("Error creating output structure: ", err)
	}
	return nil
}

func CreateMonitoringOutput(v *MetaDataVectorWrapper) error {
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

			for queryName, vectors := range queryNames {
				queryFolder := fmt.Sprintf("%s/%s", sourceFolder, queryName)
				CreateOutputFolder(queryFolder)

				queryFile := CreateFile(queryFolder, queryName+".csv")

				for _, vector := range vectors {
					for _, value := range vector {
						timestamp := value.Timestamp.Time()
						logrus.Info("Metrics are writte at time: ", timestamp)
						v.WriteToCSV(queryFile, timestamp, model.LabelSet(value.Metric), float64(value.Value))
					}
				}
			}
		}
	}
	return nil
}

func (v *MetaDataVectorWrapper) WriteToCSV(outputFile *os.File, timestamp time.Time, metricLabels model.LabelSet, value float64) error {
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

	// // Write data to the file.
	values := ReadLabelValues(timestamp, metricLabels, value)
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
	header := []string{}
	// Write timestamp as first table entry
	header = append(header, "value")
	header = append(header, "unix timestamp")
	for key := range labelNames {
		castLabels := string(key)
		header = append(header, castLabels)
	}
	// sort.Strings(header)
	return header
}

func ReadLabelValues(timestamp time.Time, labelValues model.LabelSet, value float64) []string {
	record := []string{
		timestamp.String(),
		labelValues.String(),
		fmt.Sprintf("%f", value),
	}
	return record
}

func GetFileIdentifier(sample model.Sample) string {
	return sample.Value.String()
}

func GetTimeStamp(sample model.Sample) time.Time {
	return sample.Timestamp.Time()
}
