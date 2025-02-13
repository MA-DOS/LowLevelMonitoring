package watcher

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type NextflowContainer struct {
	Name        string `json:"name"`
	PID         int    `json:"pid"`
	ContainerID string `json:"container_id"`
	WorkDir     string `json:"work_dir"`
}

func (c *NextflowContainer) InspectContainer(containers []types.Container) (containerData NextflowContainer) {
	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var nextflowContainers []NextflowContainer

	var nextflowContainer NextflowContainer
	re := regexp.MustCompile(`^/nxf.*`)

	for _, container := range containers {
		wg.Add(1)
		go func(container types.Container) {
			defer wg.Done()
			ctrState, err := apiClient.ContainerInspect(context.Background(), container.ID)
			if err != nil {
				fmt.Printf("Error inspecting container: %s", err)
				return
			}
			// fmt.Printf("Container ID: %s, Container Pid: %d\n", ctrState.ID, ctrState.State.Pid)
			// fmt.Printf("Container ID: %s, Container Name: %s, Container WorkDir: %s\n", ctrState.ID, container.Names[index], ctrState.GraphDriver.Data["WorkDir"])

			nextflowContainer = NextflowContainer{
				Name:        ctrState.Name,
				PID:         ctrState.State.Pid,
				ContainerID: ctrState.ID,
				WorkDir:     ctrState.Config.WorkingDir,
			}

			if re.MatchString(ctrState.Name) {
				// fmt.Println("Nextflow container found!")
				mu.Lock()
				nextflowContainers = append(nextflowContainers, nextflowContainer)
				mu.Unlock()
				// fmt.Println(ctrState.Name)
				// fmt.Println(nextflowContainer)
				// fmt.Println(ctrState.State.Pid)
			}
			// fmt.Println("Not a Nextflow container!")
		}(container)
	}

	wg.Wait()

	// Write stuff to output
	path := "results"
	fileName := "nextflow_containers.csv"
	fullPath := fmt.Sprintf("%s/%s", path, fileName)
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

	for _, container := range nextflowContainers {
		writer.Write([]string{
			container.Name,
			fmt.Sprintf("%d", container.PID),
			container.ContainerID,
			container.WorkDir,
		})
	}
	return containerData
}

func (c *NextflowContainer) ListContainers() []types.Container {
	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	containers, err := apiClient.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		panic(err)
	}
	return containers
}
