package watcher

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type WorkflowTask struct {
	NameToID map[string]int
	// Started  string
	// Exited   string
	// Workdir  string
}

func GetWorkflowRun(f string) bool {
	var registerNewWorkflow bool
	var registerRunningWorkflow bool

	file, err := os.Open(f)
	if err != nil {
		logrus.Error("Error opening file")
	}
	defer file.Close()

	todayHourMinute := time.Now().Truncate(time.Minute).Format("Jan-02 15:04")
	todayHour := time.Now().Truncate(time.Hour).Format("Jan-02 15")
	fmt.Println(todayHourMinute)
	fmt.Println(todayHour)

	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	if err != nil {
		logrus.Error("Error reading file")
	}
	if strings.Contains(line, todayHourMinute) {
		logrus.Info("Found today's date in Nextflow logs!")
		registerNewWorkflow = true
	} else if strings.Contains(line, todayHour) {
		logrus.Info("No new workflow detected")
		registerRunningWorkflow = true
	} else {
		logrus.Info("Nextflow log does not hold a Workflow run for the kcurrent time...")
		registerNewWorkflow = false
		registerRunningWorkflow = false
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(1 * time.Second)
				continue
			}
			logrus.Error("Error reading file")
			return false
		}
		if strings.Contains(line, "Workflow started") && registerNewWorkflow {
			logrus.Info("New Workflow Run detected")
			return true
		} else if registerRunningWorkflow {
			logrus.Info("Workflow is maybe running or needs to be started...")
			return false
		}
	}
}

func (wft *WorkflowTask) WatchCompletedTasks(f string) error {

	file, err := os.Open(f)
	if err != nil {
		fmt.Println("Error opening file")
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	re := regexp.MustCompile(`Task completed > TaskHandler\[jobId: (\d+);.*?name: ([^\(]+)`)

	// Keep the file pointer at the current position
	for {
		wft.NameToID = make(map[string]int)
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(1 * time.Second)
				continue
			}
			return fmt.Errorf("error reading file: %w", err)
		}

		if strings.Contains(line, "[Task monitor] DEBUG") {
			matches := re.FindStringSubmatch(line)
			if len(matches) < 3 {
				// logrus.Warnf("No match for line: %s", line) // Debugging output
				continue
			}
			// Extract the regex matches
			jobID := matches[1]
			taskName := matches[2]
			// started := matches[4]
			// exited := matches[5]
			// workdir := matches[3]

			// Instantiate a new WorkflowTask Object for each completed task
			wft.NameToID[taskName], _ = strconv.Atoi(jobID)
			// wft.Started = started
			// wft.Exited = exited
			// wft.Workdir = workdir
			// logrus.Infof("Found	Task %s with job ID %s as COMPLETED", taskName, jobID)
			// fmt.Print(wft.NameToID)
		}
	}
}
