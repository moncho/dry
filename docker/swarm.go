package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	pkgError "github.com/pkg/errors"
)

//SwarmNodes returns the nodes that are part of the Swarm
func (daemon *DockerDaemon) SwarmNodes() ([]swarm.Node, error) {

	nodes, err := daemon.client.NodeList(context.Background(), types.NodeListOptions{})
	if err == nil {
		return nodes, nil
	}
	return nil, pkgError.Wrap(err, "Error retrieving node list")
}
