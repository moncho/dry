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
	return []swarm.Node{swarm.Node{}}, nil
}
