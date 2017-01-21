package mock

import (
	"strconv"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	dockerAPI "github.com/docker/docker/client"
)

//Docker repo is vendoring x/net/context and it seems that it conflicts
//with whatever version of the same package the vendoring tools retrieve
//A way to fix this is by removing the vendored package from the docker
//directory of the vendor tool of dry, so:
//rm -rf vendor/github.com/docker/docker/vendor/golang.org/x/

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
