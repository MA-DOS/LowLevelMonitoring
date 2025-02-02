package watcher

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type NextflowContainer struct {
	ContainerID string `json:"container_id"`
	PID         int    `json:"pid"`
	Name        string `json:"name"`
}

func (c *NextflowContainer) InspectContainer(containers []types.Container) (container NextflowContainer) {
	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	var nextflowContainer NextflowContainer

	for _, container := range containers {
		ctrState, err := apiClient.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			panic(err)
		}
		// fmt.Printf("Container ID: %s, Container Pid: %d\n", ctrState.ID, ctrState.State.Pid)
		nextflowContainer = NextflowContainer{
			ContainerID: ctrState.ID,
			PID:         ctrState.State.Pid,
			Name:        ctrState.Name,
		}
	}
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
