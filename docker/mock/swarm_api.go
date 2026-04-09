package mock

import (
	"context"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

const (
	// LabelNamespace is the label used to track stack resources
	// Copied from https://github.com/docker/cli/blob/master/cli/compose/convert/compose.go
	LabelNamespace = "com.docker.stack.namespace"
)

// SwarmAPIClientMock mocks docker SwarmAPIClient
type SwarmAPIClientMock struct {
	client.APIClient
}

// NodeList returns a list with one Node
func (mock SwarmAPIClientMock) NodeList(context.Context, client.NodeListOptions) (client.NodeListResult, error) {
	return client.NodeListResult{
		Items: []swarm.Node{{
			ID: "1",
		}},
	}, nil
}

// TaskList returns a list of tasks, node with id 1 will return a non empty list
func (mock SwarmAPIClientMock) TaskList(_ context.Context, options client.TaskListOptions) (client.TaskListResult, error) {
	if f := options.Filters["node"]; f["1"] {
		return client.TaskListResult{
			Items: []swarm.Task{
				{
					ID:     "1",
					NodeID: "1",
				},
			},
		}, nil
	}

	return client.TaskListResult{}, nil
}

// ServiceList returns a list of services
func (mock SwarmAPIClientMock) ServiceList(context.Context, client.ServiceListOptions) (client.ServiceListResult, error) {
	return client.ServiceListResult{Items: []swarm.Service{
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
	}}, nil
}

// ConfigList mock
func (mock SwarmAPIClientMock) ConfigList(context.Context, client.ConfigListOptions) (client.ConfigListResult, error) {
	return client.ConfigListResult{}, nil
}

// NetworkList mock
func (mock SwarmAPIClientMock) NetworkList(context.Context, client.NetworkListOptions) (client.NetworkListResult, error) {
	return client.NetworkListResult{}, nil
}

// SecretList mock
func (mock SwarmAPIClientMock) SecretList(context.Context, client.SecretListOptions) (client.SecretListResult, error) {
	return client.SecretListResult{}, nil
}
