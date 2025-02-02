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
	var registerWorkflow bool

	file, err := os.Open(f)
	if err != nil {
		fmt.Println("Error opening file")
	}
	defer file.Close()

	// TODO: Date detection does not work.
	today := time.Now().Truncate(time.Minute).Format("Jan-02 15:04")
	fmt.Print(today)

	logTime := bufio.NewScanner(file)
	for logTime.Scan() {
		line := logTime.Text()
		if strings.Contains(line, today) {
			fmt.Print(line)
			registerWorkflow = true
		} else {
			registerWorkflow = false
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Workflow started") && registerWorkflow {
				logrus.Info("New Workflow Run detected")
				break
			}
		}
		if err := scanner.Err(); err != nil {
			logrus.Error("Error reading file")
			return false
		}
	}
	return true
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
			fmt.Print(wft.NameToID)
		}
	}
}
