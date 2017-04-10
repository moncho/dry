package docker

import (
	"testing"

	"github.com/moncho/dry/docker/mock"
)

func TestSwarmNodeRetrieval(t *testing.T) {
	daemon := DockerDaemon{client: mock.SwarmAPIClientMock{}}
	nodes, err := daemon.SwarmNodes()

	if err != nil {
		t.Errorf("Retrieving the list of swarm nodes resulted in error: %s", err.Error())
	}
	if len(nodes) != 1 {
		t.Errorf("Expected a list with one node, got %d", len(nodes))

	}
}
