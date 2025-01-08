package aggregate

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

type MetaDataVectorWrapper struct {
	Vector     model.Vector
	MetricsMap map[string][]model.Vector
}

func NewMetaDataVectorWrapper(v model.Vector, m map[string][]model.Vector) *MetaDataVectorWrapper {
	return &MetaDataVectorWrapper{
		Vector:     v,
		MetricsMap: m,
	}
}

// TODO: Need to refactor this function to be more modular.
// This func is called on a MetaDataVectorWrapper object so it can access the fileds of the struct.
func (v *MetaDataVectorWrapper) CreateMetaDataOutput() error {
	var timestamp time.Time
	for key, vectors := range v.MetricsMap {
		// Create a new folder for the key (monitoring target)
		targetFolder := fmt.Sprintf("results/%s", key)
		err := CreateOutputFolders(targetFolder)
		if err != nil {
			if os.IsExist(err) {
				logrus.Warn("Folder already exists: ", targetFolder)
			} else {
				logrus.Error("Error creating folder: ", err)
				return err
			}
		}

		for _, vector := range vectors {
			for _, value := range vector {
				fileIdentifier := GetFileIdentifier(*value)
				fileName := GetFileName(targetFolder, fileIdentifier)
				timestamp = GetTimeStamp(*value)
				fmt.Print("Timestamp: ", timestamp)

				// Create or open file in append mode
				file, err := CreateFile(fileName)
				if err != nil {
					return err
				}
				logrus.Info("File opened: ", fileName)

				// Calling helper to write to file
				err = v.WriteToCSV(file, timestamp, model.LabelSet(value.Metric))
				if err != nil {
					logrus.Error("Error writing to file: ", err)
				}
				file.Close() // Close the file after writing
			}
		}
	}
	return nil
}

func ReadHeaderFields(labelNames model.LabelSet) []string {
	header := []string{}
	// Write timestamp as first table entry
	header = append(header, "unix timestamp")
	for key := range labelNames {
		castLabels := string(key)
		header = append(header, castLabels)
	}
	sort.Strings(header)
	return header
}

func ReadLabelValues(timestamp time.Time, labelValues model.LabelSet) []string {
	// values := []string{}
	// _ = append(values, timestamp.String())
	// for _, value := range labelValues {
	// 	castValue := string(value)
	// 	values = append(values, castValue)
	// }
	// sort.Strings(values)
	// return values

	record := []string{
		timestamp.String(),
		labelValues.String(),
	}
	return record
}

func (v *MetaDataVectorWrapper) WriteToCSV(outputFile *os.File, timestamp time.Time, metricLabels model.LabelSet) error {
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
	// header := ReadHeaderFields(metricLabels)
	// if err := w.Write(header); err != nil {
	// 	logrus.Error("Error writing the header to CSV: ", err)
	// 	return err
	// }

	// Write data to the file.
	values := ReadLabelValues(timestamp, metricLabels)
	if err := w.Write(values); err != nil {
		logrus.Error("Error writing to CSV")
		return err
	}
	return nil
}

// TODO: This might be built into create output function.
// func GetFolderNames(m *map[string][]model.Vector) string {
// 	var folderName string
// 	for key := range *m {
// 		targetFolder := fmt.Sprintf("results/%s", key)
// 		folderName = targetFolder
// 	}
// 	return folderName
// }

// Helper to create folder.
// TODO: Do this for each metric type in a go routine
func CreateOutputFolders(folderName string) error {
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		err := os.Mkdir(folderName, 0755)
		if err != nil {
			logrus.Error("Error creating folder: ", err)
		}
	}
	return nil
}

func GetFileIdentifier(sample model.Sample) string {
	return sample.Value.String()
}

func GetTimeStamp(sample model.Sample) time.Time {
	return sample.Timestamp.Time()
}

// Helper to get file name.
func GetFileName(targetFolder, fileIdentifier string) string {
	return fmt.Sprintf("%s/%s.csv", targetFolder, fileIdentifier)
}

func CreateFile(fileName string) (*os.File, error) {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error("Error opening file: ", err)
		return nil, err
	}
	return file, nil
}

// TODO: Helper functions are needed to split this.
// func (v *MetaDataVectorWrapper) CreateMetaDataOutput(m *map[string][]model.Vector) error {
// 	for key := range *m {
// 		// Create a new folder for the key (monitoring target)
// 		targetFolder := fmt.Sprintf("results/%s", key)
// 		err := os.Mkdir(targetFolder, 0755)
// 		if err != nil {
// 			if os.IsExist(err) {
// 				logrus.Warn("Folder already exists: ", targetFolder)
// 			} else {
// 				logrus.Error("Error creating folder: ", err)
// 			}
// 			for _, vector := range (*m)[key] {
// 				for _, value := range vector {
// 					metrics := model.LabelSet(value.Metric)
// 					fileIdentifier := value.Value.String()
// 					fileName := fmt.Sprintf("results/%s/%s.txt", key, fileIdentifier)
// 					file, err := os.Create(fileName)
// 					// Calling helper to write to file
// 					err = v.WriteToCSV(file, metrics)
// 					logrus.Info("Writing to file: ", fileName)
// 					if err != nil {
// 						if os.IsExist(err) {
// 							logrus.Warn("File already exists: ", fileName)
// 						} else {
// 							logrus.Error("Error creating file: ", err)
// 						}
// 						defer file.Close()
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }
