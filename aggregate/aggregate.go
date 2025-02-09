package aggregate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const resultPath = "results"

// Walking down the results directory and matching metric entities
func MatchResults(r string) {
	err := filepath.Walk(resultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Error("Error walking the path")
			return err
		}
		fmt.Println(path)
		return nil
	})
	if err != nil {
		logrus.Error("Error walking the path")
	}
}
