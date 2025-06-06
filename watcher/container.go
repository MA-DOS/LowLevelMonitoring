package watcher

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// Regex to match Nextflow container names.
var re = regexp.MustCompile(`^/nxf-[a-zA-Z0-9-]+$`)

type NextflowContainer struct {
	WorkerIP       string    `json:"node"`
	ContainerEvent string    `json:"event"`
	StartTime      time.Time `json:"start_time"`
	DieTime        time.Time `json:"die_time"`
	Name           string    `json:"name"`
	LifeTime       string    `json:"life_time"`
	PID            int       `json:"pid"`
	ContainerID    string    `json:"container_id"`
	WorkDir        string    `json:"work_dir"`
}

func (c *NextflowContainer) GetContainerEvents(containerEventChannel chan<- NextflowContainer) {
	// Container Client.
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.49"))
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	eventChan, errChan := apiClient.Events(context.Background(), events.ListOptions{})

	processedStarts := make(map[string]bool) // Track started containers
	processedDies := make(map[string]bool)   // Track died containers
	containerPIDs := make(map[string]int)    // Track container PIDs
	var mu sync.Mutex
	wg := sync.WaitGroup{}

	go func() {
		for {
			select {
			case event := <-eventChan:
				if event.Type == events.ContainerEventType {
					switch event.Action {
					case "start":
						processContainerEvent(event, apiClient, re, &mu, processedStarts, containerPIDs, containerEventChannel, true, &wg)
					case "die":
						processContainerEvent(event, apiClient, re, &mu, processedDies, containerPIDs, containerEventChannel, false, &wg)
					}
				}
			case err := <-errChan:
				if err != nil {
					logrus.Error("Error while watching for events: ", err)
				}
			}
		}
	}()
	wg.Wait()
}

func processContainerEvent(event events.Message, apiClient *client.Client, re *regexp.Regexp, mu *sync.Mutex, processed map[string]bool, containerPIDs map[string]int, containerEventChannel chan<- NextflowContainer, isStartEvent bool, wg *sync.WaitGroup) {
	mu.Lock()
	if processed[event.Actor.ID] {
		mu.Unlock()
		return
	}
	processed[event.Actor.ID] = true
	mu.Unlock()

	go func() {
		// Get container metadata for prometheus queries.
		containerInfo, err := apiClient.ContainerInspect(context.Background(), event.Actor.ID)
		if err != nil {
			logrus.Printf("Error inspecting container %s: %v", event.Actor.ID, err)
			return
		}

		if len(containerInfo.Name) > 0 && re.MatchString(containerInfo.Name) {
			eventType := "[STARTED]"
			if !isStartEvent {
				eventType = "[DIED]"
			}
			logrus.Infof("%s nextflow container: %s\n", eventType, containerInfo.Name)
			pid := containerInfo.State.Pid

			// wg.Add(1)
			// go func() {
			// 	defer wg.Done()
			// 	getContainerStats(apiClient, containerInfo.ID, containerInfo.Name)
			// }()
			wg.Add(1)
			go func() {
				defer wg.Done()
				getContainerStatsManual(apiClient, containerInfo.ID, containerInfo.Name)
			}()

			if !isStartEvent {
				mu.Lock()
				pid = containerPIDs[event.Actor.ID]
				if pid == 0 {
					logrus.Warn("Container Process interrupted, PID not found")
					return
				}
				mu.Unlock()
			}

			nextflowContainer := createNextflowContainer(containerInfo, pid)

			if isStartEvent {
				mu.Lock()
				containerPIDs[event.Actor.ID] = pid
				mu.Unlock()
				WriteStartedToOutput(nextflowContainer)
			} else {
				mu.Lock()
				pid = containerPIDs[event.Actor.ID]
				if pid == 0 {
					logrus.Warn("Container Process interrupted, PID not found")
					return
				}
				mu.Unlock()
				containerEventChannel <- nextflowContainer
				WriteDiedToOutput(nextflowContainer)
			}
		}
	}()
}

func getContainerStats(apiClient *client.Client, containerID, containerName string) {
	ctx := context.Background()
	containerStats, err := apiClient.ContainerStats(ctx, containerID, true)
	if err != nil {
		logrus.Errorf("Error inspecting container %s: %v", containerID, err)
		return
	}
	defer containerStats.Body.Close()

	if _, err := os.Stat("results"); os.IsNotExist(err) {
		logrus.Errorf("Results directory does not exist, not writing stats for %s", containerName)
		return
	}
	statsFileName := fmt.Sprintf("results/%s.json", containerName)
	// statsFile, err := os.Create(statsFileName)
	statsFile, err := os.OpenFile(statsFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Errorf("Error creating stats file: %v", err)
	} else {
		defer statsFile.Close()
		decoder := json.NewDecoder(containerStats.Body)
		encoder := json.NewEncoder(statsFile)

		for {
			var stats container.StatsResponse
			if err := decoder.Decode(&stats); err != nil {
				if err == io.EOF {
					break
				}
				logrus.Errorf("Error decoding container stats: %v", err)
				break
			}
			if err := encoder.Encode(stats); err != nil {
				logrus.Errorf("Error writing stats to file: %v", err)
				break
			}
		}
	}
	containerStats.Body.Close()
}

func getContainerStatsManual(apiClient *client.Client, containerID, containerName string) {
	ctx := context.Background()
	statsFileName := fmt.Sprintf("results/%s.json", containerName)
	statsFile, err := os.OpenFile(statsFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Errorf("Error creating stats file: %v", err)
		return
	}
	defer statsFile.Close()
	encoder := json.NewEncoder(statsFile)

	for {
		containerStats, err := apiClient.ContainerStats(ctx, containerID, false)
		if err != nil {
			logrus.Errorf("Error inspecting container %s: %v", containerID, err)
			break
		}
		var stats container.StatsResponse
		decoder := json.NewDecoder(containerStats.Body)
		if err := decoder.Decode(&stats); err != nil {
			containerStats.Body.Close()
			logrus.Errorf("Error decoding container stats: %v", err)
			break
		}
		containerStats.Body.Close()
		if err := encoder.Encode(stats); err != nil {
			logrus.Errorf("Error writing stats to file: %v", err)
			break
		}

		// Check if container is still running
		inspect, err := apiClient.ContainerInspect(ctx, containerID)
		if err != nil || !inspect.State.Running {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func createNextflowContainer(containerInfo types.ContainerJSON, pid int) NextflowContainer {
	return NextflowContainer{
		StartTime: func() time.Time {
			t, _ := time.Parse(time.RFC3339, containerInfo.State.StartedAt)
			return t
		}(),
		DieTime: func() time.Time {
			t, _ := time.Parse(time.RFC3339, containerInfo.State.FinishedAt)
			return t
		}(),
		Name: strings.TrimPrefix(containerInfo.Name, "/"),
		LifeTime: func() string {
			startTime, _ := time.Parse(time.RFC3339, containerInfo.State.StartedAt)
			dieTime, _ := time.Parse(time.RFC3339, containerInfo.State.FinishedAt)
			return dieTime.Sub(startTime).String()
		}(),
		PID:         pid,
		ContainerID: containerInfo.ID,
		WorkDir:     containerInfo.Config.WorkingDir,
	}
}

func WriteStartedToOutput(container NextflowContainer) {
	fullPath := prepareOutputFile("results", "started_nextflow_containers.csv")
	if fullPath == "" {
		return
	}

	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error("Error opening file: ", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header if the file is empty
	if isFileEmpty(file) {
		if err := writer.Write([]string{"Name", "PID", "ContainerID", "WorkDir"}); err != nil {
			logrus.Error("Error writing CSV header: ", err)
			return
		}
	}

	// Write container data to CSV
	if err := writer.Write([]string{
		container.Name,
		fmt.Sprintf("%d", container.PID),
		container.ContainerID,
		container.WorkDir,
	}); err != nil {
		logrus.Error("Error writing container data to CSV: ", err)
	}
}

func WriteDiedToOutput(container NextflowContainer) {
	fullPath := prepareOutputFile("results", "died_nextflow_containers.csv")
	if fullPath == "" {
		return
	}

	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error("Error opening file: ", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header if the file is empty
	if isFileEmpty(file) {
		if err := writer.Write([]string{"Name", "PID", "ContainerID", "WorkDir", "LifeTime"}); err != nil {
			logrus.Error("Error writing CSV header: ", err)
			return
		}
	}

	// Write container data to CSV
	if err := writer.Write([]string{
		container.Name,
		fmt.Sprintf("%d", container.PID),
		container.ContainerID,
		container.WorkDir,
		container.LifeTime,
	}); err != nil {
		logrus.Error("Error writing container data to CSV: ", err)
	}
}

func prepareOutputFile(path, fileName string) string {
	fullPath := fmt.Sprintf("%s/%s", path, fileName)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			logrus.Error("Error creating results directory: ", err)
			return ""
		}
	}

	return fullPath
}

func isFileEmpty(file *os.File) bool {
	fileInfo, err := file.Stat()
	if err != nil {
		logrus.Error("Error getting file info: ", err)
		return false
	}
	return fileInfo.Size() == 0
}

func EscapeContainerName(containerName string) string {
	// Remove the leading '/' if present
	containerName = strings.TrimPrefix(containerName, "/")
	// Escape remaining '/' characters
	return fmt.Sprintf("Cleaned Container Name for Query: %s", strings.ReplaceAll(containerName, "/", `\/`))
}
