package docker

import (
	"testing"

	"github.com/moncho/dry/docker/mock"
)

func TestSwarmNodeRetrieval(t *testing.T) {
	daemon := DockerDaemon{client: mock.SwarmAPIClientMock{}}
	nodes, err := daemon.Nodes()

	if err != nil {
		t.Errorf("Retrieving the list of swarm nodes resulted in error: %s", err.Error())
	}
	if len(nodes) != 1 {
		t.Errorf("Expected a list with one node, got %d", len(nodes))
	}
}

func TestTaskRetrieval(t *testing.T) {
	daemon := DockerDaemon{client: mock.SwarmAPIClientMock{}}
	tasks, err := daemon.Tasks("1")

	if err != nil {
		t.Errorf("Retrieving the list of task of node 1 resulted in error: %s", err.Error())
	}
	if len(tasks) != 1 {
		t.Errorf("Expected a list with one task, got %d", len(tasks))
	}

	tasks, err = daemon.Tasks("Nope")

	if err != nil {
		t.Errorf("Retrieving the list of task of non-existing node resulted in error: %s", err.Error())
	}
	if len(tasks) != 0 {
		t.Errorf("Expected a list with no task, got %d", len(tasks))
	}
}
