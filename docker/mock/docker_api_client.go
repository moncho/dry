package mock

import (
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"strconv"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	dockerAPI "github.com/docker/docker/client"
)

//Docker repo is vendoring x/net/context and it seems that it conflicts
//with whatever version of the same package the vendoring tools retrieve
//A way to fix this is by removing the vendored package from the docker
//directory of the vendor tool of dry, so:
//rm -rf vendor/github.com/moby/moby/vendor/golang.org/x/net

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
	dockerAPI.APIClient
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

//ContainerInspect returns an empty inspection result.
func (m ContainerAPIClientMock) ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}

//ContainerCreate mocks container creation
func (mock ImageAPIClientMock) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *v1.Platform, containerName string) (container.ContainerCreateCreatedBody, error) {
	return container.ContainerCreateCreatedBody{ID: "NewContainer"}, nil
}

//ContainerStart mock, accepts everything without complains
func (mock ImageAPIClientMock) ContainerStart(ctx context.Context, container string, options types.ContainerStartOptions) error {
	return nil
}

//InspectImage mock
func (mock ImageAPIClientMock) InspectImage(ctx context.Context, image string) (types.ImageInspect, error) {
	return types.ImageInspect{
		ContainerConfig: &container.Config{},
	}, nil
}

//ImageInspectWithRaw mock
func (mock ImageAPIClientMock) ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error) {
	return types.ImageInspect{
		ContainerConfig: &container.Config{},
	}, nil, nil
}
