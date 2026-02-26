package docker

import (
	"strconv"
	"testing"

	"github.com/docker/docker/api/types/container"
	dockerAPI "github.com/docker/docker/client"
	"github.com/moncho/dry/docker/mock"
)

func TestContainerListRetrieval(t *testing.T) {
	c, _ := containers(createClient())

	for i, container := range c {
		if container.ID != strconv.Itoa(i) {
			t.Errorf("Invalid container %v", container)
		}
	}
}

func TestContainerConversionToPointerList(t *testing.T) {
	var containers []container.Summary

	for i := 0; i < 10; i++ {
		containers = append(containers, container.Summary{
			ID: strconv.Itoa(i),
		})
	}
	var cPointers []*container.Summary
	for i := range containers {
		cPointers = append(cPointers, &containers[i])
	}

	for i, container := range cPointers {
		if container.ID != strconv.Itoa(i) {
			t.Errorf("Invalid container %v", container)
		}
	}
}
func createClient() dockerAPI.ContainerAPIClient {
	return mock.ContainerAPIClientMock{}
}
