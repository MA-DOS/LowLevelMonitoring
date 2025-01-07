package aggregate

import (
	"fmt"
	"os"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

type MetaDataVectorWrapper struct {
	Vector model.Vector
}

func NewMetaDataVectorWrapper(v model.Vector) *MetaDataVectorWrapper {
	return &MetaDataVectorWrapper{
		Vector: v,
	}
}

// TODO: Do i actually need to parse something because i already got a map as return from Prometheus?
func (v *MetaDataVectorWrapper) Parse(isVector bool, valueMap map[string][]model.Vector) error {
	if isVector {

	}
	return nil
}

func (v *MetaDataVectorWrapper) Write(m *map[string][]model.Value) error {
	for key, _ := range *m {
		// Create a new folder for the key (monitoring target)
		targetFolder := fmt.Sprintf("results/%s", key)
		err := os.Mkdir(targetFolder, 0755)
		if err != nil {
			if os.IsExist(err) {
				logrus.Warn("Folder already exists: ", targetFolder)
			} else {
				logrus.Error("Error creating folder: ", err)
			}
			for _, value := range (*m)[key] {
				metric := value.(model.Vector)
				fileIdentifier := metric.String()
				fileName := fmt.Sprintf("results/%s/%s.txt", key, fileIdentifier)
			}
		}
	}
	return nil
}

// TODO: Do i actually need this or can I just work with model.Value?
func CheckAndAssertType(promResult map[string][]model.Value) (bool, string, error) {
	var typeOfResult bool
	var result string
	for key, values := range promResult {
		for _, value := range values {
			switch value.Type() {
			case model.ValMatrix:
				matrix := value.(model.Matrix)
				_ = matrix
				result += fmt.Sprintf("Key: %s, Type: Matrix\n", key)
			case model.ValVector:
				vector := value.(model.Vector)
				_ = vector
				result += fmt.Sprintf("Key: %s, Type: Vector\n", key)
				typeOfResult = true
			case model.ValScalar:
				scalar := value.(*model.Scalar)
				_ = scalar
				result += fmt.Sprintf("Key: %s, Type: Scalar\n", key)
			case model.ValString:
				str := value.(*model.String)
				_ = str
				result += fmt.Sprintf("Key: %s, Type: String\n", key)
			default:
				result += fmt.Sprintf("Key: %s, Unknown type\n", key)

			}
		}
	}
	return typeOfResult, result, nil
}
