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

//ContainerAPIClientMock mocks docker ContainerAPIClient
type ContainerAPIClientMock struct {
	dockerAPI.ContainerAPIClient
	Containers []types.Container
}

//NetworkAPIClientMock mocks docker NetworkAPIClient
type NetworkAPIClientMock struct {
	dockerAPI.NetworkAPIClient
}

//ImageAPIClientMock mocks docker ImageAPIClient
type ImageAPIClientMock struct {
	dockerAPI.ImageAPIClient
}

//ContainerList returns a list with 10 container with IDs from 0 to 9.
func (m ContainerAPIClientMock) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	if len(m.Containers) > 0 {
		return m.Containers, nil
	}
	var containers []types.Container

	for i := 0; i < 10; i++ {
		containers = append(containers, types.Container{
			ID: strconv.Itoa(i),
		})
	}

	return containers, nil
}
