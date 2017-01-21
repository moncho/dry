package mock

import (
	"strconv"

	dockerAPI "github.com/docker/docker/api/client"
	"github.com/docker/docker/api/types"
	context "golang.org/x/net/context"
)

//APIClientMock mocks docker APIClient
type APIClientMock struct {
	dockerAPI.APIClient
}

//ContainerList returns a list with 10 container with IDs from 0 to 9.
func (m APIClientMock) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	var containers []types.Container

	for i := 0; i < 10; i++ {
		containers = append(containers, types.Container{
			ID: strconv.Itoa(i),
		})
	}

	return containers, nil
}
