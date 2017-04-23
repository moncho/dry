package mock

import (
	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	dockerAPI "github.com/docker/docker/client"
)

//SwarmAPIClientMock mocks docker SwarmAPIClient
type SwarmAPIClientMock struct {
	dockerAPI.APIClient
}

//NodeList returns a list with one Node
func (mock SwarmAPIClientMock) NodeList(context context.Context, options types.NodeListOptions) ([]swarm.Node, error) {
	return []swarm.Node{swarm.Node{
		ID: "1",
	}}, nil
}

//TaskList returns a list of tasks, node with id 1 will return a non empty list
func (mock SwarmAPIClientMock) TaskList(context context.Context, options types.TaskListOptions) ([]swarm.Task, error) {
	nodeID := options.Filters.Get("node")[0]
	if nodeID == "1" {
		return []swarm.Task{
			swarm.Task{
				ID:     "1",
				NodeID: "1",
			},
		}, nil
	}

	return nil, nil
}
