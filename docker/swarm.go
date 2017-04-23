package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	pkgError "github.com/pkg/errors"
)

//Node returns the node with the given id
func (daemon *DockerDaemon) Node(id string) (*swarm.Node, error) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	node, _, err := daemon.client.NodeInspectWithRaw(ctx, id)
	if err == nil {
		return &node, nil
	}
	return nil, pkgError.Wrapf(err, "Error retrieving node with id %s", id)
}

//Nodes returns the nodes that are part of the Swarm
func (daemon *DockerDaemon) Nodes() ([]swarm.Node, error) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	nodes, err := daemon.client.NodeList(ctx, types.NodeListOptions{})
	if err == nil {
		return nodes, nil
	}
	return nil, pkgError.Wrap(err, "Error retrieving node list")
}

//Tasks returns the nodes that are part of the Swarm
func (daemon *DockerDaemon) Tasks(nodeID string) ([]swarm.Task, error) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	filter := filters.NewArgs()
	filter.Add("node", nodeID)

	nodeTasks, err := daemon.client.TaskList(ctx, types.TaskListOptions{Filters: filter})

	if err == nil {
		return nodeTasks, nil
	}
	return nil, pkgError.Wrap(err, "Error retrieving task list")
}
