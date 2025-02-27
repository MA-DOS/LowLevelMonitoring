package watcher

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type NextflowContainer struct {
	StartTime   string `json:"start_time"`
	DieTime     string `json:"die_time"`
	Name        string `json:"name"`
	PID         int    `json:"pid"`
	ContainerID string `json:"container_id"`
	WorkDir     string `json:"work_dir"`
}

// func (c *NextflowContainer) GetContainerEvents(containerEventChannel chan<- NextflowContainer) (containerStartTime string) {
// 	// Channel to communicate container events to main monitoring thread.
// 	var nextflowContainer NextflowContainer

// 	re := regexp.MustCompile(`^/nxf-[a-zA-Z0-9-]+$`)

// 	// Container Client.
// 	apiClient, err := client.NewClientWithOpts(client.FromEnv)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer apiClient.Close()

// 	eventChan, errChan := apiClient.Events(context.Background(), events.ListOptions{})

// 	logrus.Info("[NEXTFLOW NOT RUNNING]...")

// 	for {
// 		select {
// 		case event := <-eventChan:
// 			if event.Type == events.ContainerEventType && (event.Action == "start" /*|| event.Action == "create"*/) {
// 				containerInfo, err := apiClient.ContainerInspect(context.Background(), event.Actor.ID)
// 				if err != nil {
// 					log.Printf("Error inspecting container %s: %v", event.Actor.ID, err)
// 					continue
// 				}
// 				// if len(containerInfo.Name) > 0 && re.MatchString(EscapeContainerName(containerInfo.Name)) {
// 				if len(containerInfo.Name) > 0 && re.MatchString(containerInfo.Name) {
// 					logrus.Infof("Found matching container: %s\n with regex: %s\n", containerInfo.Name, re)
// 					// containerJson, _ := json.MarshalIndent(containerInfo, "", "    ")
// 					logrus.Infof("[STARTED] nextflow container: %s spawned...!\n", containerInfo.Name)
// 					nextflowContainer = NextflowContainer{
// 						StartTime:   containerInfo.State.StartedAt,
// 						DieTime:     containerInfo.State.FinishedAt,
// 						Name:        (containerInfo.Name),
// 						PID:         containerInfo.State.Pid,
// 						ContainerID: containerInfo.ID,
// 						WorkDir:     containerInfo.Config.WorkingDir,
// 					}
// 					// fmt.Println(nextflowContainer)
// 					WriteToOutput(nextflowContainer)
// 					// containerEventChannel <- nextflowContainer.StartTime
// 				}
// 			}
// 		case event := <-eventChan:
// 			if event.Type == events.ContainerEventType && event.Action == "die" {
// 				containerInfo, err := apiClient.ContainerInspect(context.Background(), event.Actor.ID)
// 				if err != nil {
// 					log.Printf("Error inspecting container %s: %v", event.Actor.ID, err)
// 					continue
// 				}
// 				// if len(containerInfo.Name) > 0 && re.MatchString(EscapeContainerName(containerInfo.Name)) {
// 				if len(containerInfo.Name) > 0 && re.MatchString(containerInfo.Name) {
// 					logrus.Infof("Found matching container: %s\n with regex: %s\n", containerInfo.Name, re)
// 					logrus.Infof("[DIED] nextflow container: %s died...!\n", containerInfo.Name)

// 					containerEventChannel <- NextflowContainer{
// 						StartTime: containerInfo.State.StartedAt,
// 						DieTime:   containerInfo.State.FinishedAt,
// 						// Name:      EscapeContainerName(containerInfo.Name),
// 						Name: (containerInfo.Name),
// 					}
// 					containerEventChannel <- nextflowContainer
// 				}
// 			}
// 		case err := <-errChan:
// 			logrus.Error("Error while watching for events: ", err)
// 		}
// 	}
// }

// func (c *NextflowContainer) GetContainerEvents(containerEventChannel chan<- NextflowContainer) {
// 	// Channel to communicate container events to main monitoring thread.
// 	re := regexp.MustCompile(`^/nxf-[a-zA-Z0-9-]+$`)

// 	// Container Client.
// 	apiClient, err := client.NewClientWithOpts(client.FromEnv)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer apiClient.Close()

// 	eventChan, errChan := apiClient.Events(context.Background(), events.ListOptions{})

// 	logrus.Info("[NEXTFLOW NOT RUNNING]...")
// 	processedContainers := make(map[string]bool)

// 	for {
// 		select {
// 		case event := <-eventChan:
// 			if event.Type == events.ContainerEventType {
// 				// containerInfo, err := apiClient.ContainerInspect(context.Background(), event.Actor.ID)
// 				// if err != nil {
// 				// 	log.Printf("Container %s: %v", event.Actor.ID, err)
// 				// 	continue
// 				// }

// 				// Attempt to inspect the container
// 				containerInfo, err := apiClient.ContainerInspect(context.Background(), event.Actor.ID)
// 				if err != nil {
// 					if client.IsErrNotFound(err) {
// 						logrus.Warnf("Container %s not found: %v", event.Actor.ID, err)
// 					} else {
// 						log.Printf("Error inspecting container %s: %v", event.Actor.ID, err)
// 					}
// 					continue
// 				}

// 				if len(containerInfo.Name) > 0 && re.MatchString(containerInfo.Name) {
// 					// logrus.Infof("Found matching container: %s\n with regex: %s\n", containerInfo.Name, re)
// 					if event.Action == "start" {
// 						logrus.Infof("[STARTED] nextflow container: %s spawned...!\n", containerInfo.Name)
// 					} else if event.Action == "die" {
// 						if !processedContainers[containerInfo.ID] {
// 							logrus.Infof("[DIED] nextflow container: %s died...!\n", containerInfo.Name)
// 							nextflowContainer := NextflowContainer{
// 								StartTime:   containerInfo.State.StartedAt,
// 								DieTime:     containerInfo.State.FinishedAt,
// 								Name:        containerInfo.Name,
// 								PID:         containerInfo.State.Pid,
// 								ContainerID: containerInfo.ID,
// 								WorkDir:     containerInfo.Config.WorkingDir,
// 							}

// 							WriteToOutput(nextflowContainer)
// 							containerEventChannel <- nextflowContainer
// 							processedContainers[containerInfo.ID] = true
// 						}
// 					}
// 				}
// 			}
// 		case err := <-errChan:
// 			logrus.Error("Error while watching for events: ", err)
// 			// default:
// 			// 	logrus.Info("[NO EVENTS TO PROCESS]...")
// 		}
// 	}
// }

func (c *NextflowContainer) GetContainerEvents(containerEventChannel chan<- NextflowContainer) {
	// Regex to match Nextflow container names.
	re := regexp.MustCompile(`^/nxf-[a-zA-Z0-9-]+$`)

	// Container Client.
	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	eventChan, errChan := apiClient.Events(context.Background(), events.ListOptions{})

	logrus.Info("[NEXTFLOW NOT RUNNING]...")

	processedStarts := make(map[string]bool) // Track started containers
	processedDies := make(map[string]bool)   // Track died containers
	var mu sync.Mutex

	go func() {
		for {
			select {
			case event := <-eventChan:
				if event.Type == events.ContainerEventType {
					switch event.Action {
					case "start":
						mu.Lock()
						if processedStarts[event.Actor.ID] {
							mu.Unlock()
							continue
						}
						processedStarts[event.Actor.ID] = true
						mu.Unlock()

						go func(event events.Message) {
							containerInfo, err := apiClient.ContainerInspect(context.Background(), event.Actor.ID)
							if err != nil {
								log.Printf("Error inspecting started container %s: %v", event.Actor.ID, err)
								return
							}

							if len(containerInfo.Name) > 0 && re.MatchString(containerInfo.Name) {
								logrus.Infof("[STARTED] nextflow container: %s spawned...!\n", containerInfo.Name)
								nextflowContainer := NextflowContainer{
									StartTime:   containerInfo.State.StartedAt,
									DieTime:     containerInfo.State.FinishedAt,
									Name:        strings.TrimPrefix(containerInfo.Name, "/"),
									PID:         containerInfo.State.Pid,
									ContainerID: containerInfo.ID,
									WorkDir:     containerInfo.Config.WorkingDir,
								}

								WriteToOutput(nextflowContainer)
							}
						}(event)

					case "die":
						mu.Lock()
						if processedDies[event.Actor.ID] {
							mu.Unlock()
							continue
						}
						processedDies[event.Actor.ID] = true
						mu.Unlock()

						go func(event events.Message) {
							containerInfo, err := apiClient.ContainerInspect(context.Background(), event.Actor.ID)
							if err != nil {
								log.Printf("Error inspecting died container %s: %v", event.Actor.ID, err)
								return
							}

							if len(containerInfo.Name) > 0 && re.MatchString(containerInfo.Name) {
								logrus.Infof("[DIED] nextflow container: %s died...!\n", containerInfo.Name)
								nextflowContainer := NextflowContainer{
									StartTime:   containerInfo.State.StartedAt,
									DieTime:     containerInfo.State.FinishedAt,
									Name:        strings.TrimPrefix(containerInfo.Name, "/"),
									PID:         containerInfo.State.Pid,
									ContainerID: containerInfo.ID,
									WorkDir:     containerInfo.Config.WorkingDir,
								}

								containerEventChannel <- nextflowContainer
							}
						}(event)
					}
				}

			case err := <-errChan:
				logrus.Error("Error while watching for events: ", err)
			}
		}
	}()
}

func WriteToOutput(container NextflowContainer) {
	path := "results"
	fileName := "nextflow_containers.csv"
	fullPath := fmt.Sprintf("%s/%s", path, fileName)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			logrus.Error("Error creating results directory: ", err)
			return
		}
	}

	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Error("Error opening file: ", err)
	}

	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	fileInfo, err := file.Stat()
	if err != nil {
		return
	}
	if fileInfo.Size() == 0 {
		if err := writer.Write([]string{"Name", "PID", "ContainerID", "WorkDir"}); err != nil {
			logrus.Error("Error writing to CSV")
		}
	}

	// Write container data to CSV
	writer.Write([]string{
		container.Name,
		fmt.Sprintf("%d", container.PID),
		container.ContainerID,
		container.WorkDir,
	})
}

func EscapeContainerName(containerName string) string {
	// Remove the leading '/' if present
	containerName = strings.TrimPrefix(containerName, "/")
	// Escape remaining '/' characters
	return fmt.Sprintf("Cleaned Container Name for Query: %s", strings.ReplaceAll(containerName, "/", `\/`))
}
