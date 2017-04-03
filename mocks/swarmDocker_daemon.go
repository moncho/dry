package mocks

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

//SwarmDockerDaemon mocks a DockerDaemon operating in Swarm mode
type SwarmDockerDaemon struct {
	DockerDaemonMock
}

// Info provides a mock function with given fields:
func (_m *SwarmDockerDaemon) Info() (types.Info, error) {
	clusterInfo := swarm.ClusterInfo{ID: "MyClusterID"}
	swarmInfo := swarm.Info{LocalNodeState: swarm.LocalNodeStateActive, NodeID: "ThisNodeID", Cluster: clusterInfo}
	return types.Info{Swarm: swarmInfo}, nil
}
