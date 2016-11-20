package docker

import (
	"strconv"
	"testing"

	dockerAPI "github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/moncho/dry/docker/mock"
)

func TestContainerListRetrieval(t *testing.T) {
	c, _ := containers(createClient(), true)

	for i, container := range c {
		if container.ID != strconv.Itoa(i) {
			t.Errorf("Invalid container %v", container)
		}
	}
}

func TestContainerConversionToPointerList(t *testing.T) {
	var containers []types.Container

	for i := 0; i < 10; i++ {
		containers = append(containers, types.Container{
			ID: strconv.Itoa(i),
		})
	}
	var cPointers []*types.Container
	for i := range containers {
		cPointers = append(cPointers, &containers[i])
	}

	for i, container := range cPointers {
		if container.ID != strconv.Itoa(i) {
			t.Errorf("Invalid container %v", container)
		}
	}
}
func createClient() dockerAPI.APIClient {
	return mock.APIClientMock{}
}
