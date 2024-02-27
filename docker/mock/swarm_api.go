package mock

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	dockerAPI "github.com/docker/docker/client"
)

const (
	// LabelNamespace is the label used to track stack resources
	//Copied from https://github.com/docker/cli/blob/master/cli/compose/convert/compose.go
	LabelNamespace = "com.docker.stack.namespace"
)

// SwarmAPIClientMock mocks docker SwarmAPIClient
type SwarmAPIClientMock struct {
	dockerAPI.APIClient
}

// NodeList returns a list with one Node
func (mock SwarmAPIClientMock) NodeList(context context.Context, options types.NodeListOptions) ([]swarm.Node, error) {
	return []swarm.Node{{
		ID: "1",
	}}, nil
}

// TaskList returns a list of tasks, node with id 1 will return a non empty list
func (mock SwarmAPIClientMock) TaskList(context context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	nodeID := options.Filters.Get("node")[0]
	if nodeID == "1" {
		return []swarm.Task{
			{
				ID:     "1",
				NodeID: "1",
			},
		}, nil
	}

	return nil, nil
}

// ServiceList returns a list of services
func (mock SwarmAPIClientMock) ServiceList(context context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {

	return []swarm.Service{
		{
			ID: "1",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Labels: map[string]string{LabelNamespace: "stack1"},
				},
			},
		},
		{
			ID: "2",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Labels: map[string]string{LabelNamespace: "stack1"},
				},
			},
		},
		{
			ID: "3",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{
					Labels: map[string]string{LabelNamespace: "stack2"},
				},
			},
		},
	}, nil

}

// ConfigList mock
func (mock SwarmAPIClientMock) ConfigList(
	context context.Context, opts types.ConfigListOptions) ([]swarm.Config, error) {
	return nil, nil
}

// NetworkList mock
func (mock SwarmAPIClientMock) NetworkList(
	context context.Context, opts types.NetworkListOptions) ([]types.NetworkResource, error) {
	return nil, nil
}

// SecretList mock
func (mock SwarmAPIClientMock) SecretList(
	context context.Context, opts types.SecretListOptions) ([]swarm.Secret, error) {
	return nil, nil
}
