package mock

import (
	"context"
	"strconv"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// Docker repo is vendoring x/net/context and it seems that it conflicts
// with whatever version of the same package the vendoring tools retrieve
// A way to fix this is by removing the vendored package from the docker
// directory of the vendor tool of dry, so:
// rm -rf vendor/github.com/moby/moby/vendor/golang.org/x/net

// ContainerAPIClientMock mocks docker ContainerAPIClient
type ContainerAPIClientMock struct {
	client.ContainerAPIClient
	Containers []container.Summary
}

// NetworkAPIClientMock mocks docker NetworkAPIClient
type NetworkAPIClientMock struct {
	client.NetworkAPIClient
}

// ImageAPIClientMock mocks docker ImageAPIClient
type ImageAPIClientMock struct {
	client.APIClient
}

// ContainerList returns a list with 10 container with IDs from 0 to 9.
func (m ContainerAPIClientMock) ContainerList(ctx context.Context, options client.ContainerListOptions) (client.ContainerListResult, error) {
	if len(m.Containers) > 0 {
		return client.ContainerListResult{Items: m.Containers}, nil
	}
	var res client.ContainerListResult
	for i := 0; i < 10; i++ {
		res.Items = append(res.Items, container.Summary{
			ID: strconv.Itoa(i),
		})
	}

	return res, nil
}

// ContainerInspect returns an empty inspection result.
func (m ContainerAPIClientMock) ContainerInspect(ctx context.Context, ctr string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{}, nil
}

// ContainerCreate mocks container creation
func (mock ImageAPIClientMock) ContainerCreate(context.Context, client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	return client.ContainerCreateResult{ID: "NewContainer"}, nil
}

// ContainerStart mock, accepts everything without complains
func (mock ImageAPIClientMock) ContainerStart(context.Context, string, client.ContainerStartOptions) (client.ContainerStartResult, error) {
	return client.ContainerStartResult{}, nil
}

// InspectImage mock
func (mock ImageAPIClientMock) ImageInspect(ctx context.Context, img string, inspectOpts ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	return client.ImageInspectResult{}, nil
}
