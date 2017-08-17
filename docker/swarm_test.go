package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/moncho/dry/docker/mock"
	"github.com/pkg/errors"
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
	tasks, err := daemon.NodeTasks("1")

	if err != nil {
		t.Errorf("Retrieving the list of task of node 1 resulted in error: %s", err.Error())
	}
	if len(tasks) != 1 {
		t.Errorf("Expected a list with one task, got %d", len(tasks))
	}

	tasks, err = daemon.NodeTasks("Nope")

	if err != nil {
		t.Errorf("Retrieving the list of task of non-existing node resulted in error: %s", err.Error())
	}
	if len(tasks) != 0 {
		t.Errorf("Expected a list with no task, got %d", len(tasks))
	}
}

func TestIDResolution(t *testing.T) {
	r := &resolverMock{}
	daemon := DockerDaemon{resolver: r}
	name, err := daemon.ResolveNode("1")

	if err != nil {
		t.Errorf("Resolving node with ID 1 resulted in error: %s", err.Error())
	}
	if name != "Node1" {
		t.Errorf("Resolving node with ID 1 resulted in %s , expected %s", name, "Node1")
	}

}

type resolverMock struct {
}

func (r *resolverMock) Resolve(ctx context.Context, t interface{}, id string) (string, error) {
	switch t.(type) {
	case swarm.Node:
		return "Node" + id, nil
	case swarm.Service:
		return "Service" + id, nil
	default:
		return "", errors.Errorf("unsupported type")
	}
}
