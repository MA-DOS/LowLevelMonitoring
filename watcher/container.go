package watcher

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type NextflowContainer struct {
	Name        string `json:"name"`
	PID         int    `json:"pid"`
	ContainerID string `json:"container_id"`
	WorkDir     string `json:"work_dir"`
}

func (c *NextflowContainer) InspectContainer(containers []types.Container) (container NextflowContainer) {
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
				fmt.Println("Nextflow container found!")
				mu.Lock()
				nextflowContainers = append(nextflowContainers, nextflowContainer)
				mu.Unlock()
				// fmt.Println(ctrState.Name)
				fmt.Println(nextflowContainer)
				// fmt.Println(ctrState.State.Pid)
			}
			// fmt.Println("Not a Nextflow container!")
		}(container)
	}
	wg.Wait()
	return nextflowContainer
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
