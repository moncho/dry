package mocks

import (
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

const (
	//TestNodeID defines the ID of the swarm node for testing
	TestNodeID = "1"
)

// SwarmDockerDaemon mocks a DockerDaemon operating in Swarm mode
type SwarmDockerDaemon struct {
	DockerDaemonMock
}

// Info provides a mock function with given fields:
func (_m *SwarmDockerDaemon) Info() (types.Info, error) {
	clusterInfo := swarm.ClusterInfo{ID: "MyClusterID"}
	swarmInfo := swarm.Info{
		LocalNodeState:   swarm.LocalNodeStateActive,
		NodeID:           "ThisNodeID",
		Cluster:          &clusterInfo,
		ControlAvailable: true}
	return types.Info{
		Name:     "test",
		NCPU:     2,
		MemTotal: 1024,
		Swarm:    swarmInfo}, nil
}

// Node returns a node with the given id
func (_m *SwarmDockerDaemon) Node(id string) (*swarm.Node, error) {
	return &swarm.Node{ID: id}, nil
}

// Nodes returns a list of nodes with 1 element
func (_m *SwarmDockerDaemon) Nodes() ([]swarm.Node, error) {
	return []swarm.Node{{ID: TestNodeID}}, nil
}

// NodeTasks mock
func (_m *SwarmDockerDaemon) NodeTasks(nodeID string) ([]swarm.Task, error) {
	return []swarm.Task{{NodeID: nodeID}}, nil
}

// ResolveNode mock
func (_m *SwarmDockerDaemon) ResolveNode(id string) (string, error) {
	return strings.Join([]string{"Node", id}, ""), nil
}

// ResolveService mock
func (_m *SwarmDockerDaemon) ResolveService(id string) (string, error) {
	return strings.Join([]string{"Service", id}, ""), nil
}

// Services returns a list of services with 1 element
func (_m *SwarmDockerDaemon) Services() ([]swarm.Service, error) {
	return []swarm.Service{
		{ID: "ServiceID"}}, nil
}

// ServiceTasks returns one task per service, the task belongs to node with id "1"
func (_m *SwarmDockerDaemon) ServiceTasks(services ...string) ([]swarm.Task, error) {

	tasks := make([]swarm.Task, len(services))
	for _, service := range services {
		tasks = append(tasks, swarm.Task{ServiceID: service, NodeID: TestNodeID})
	}

	return tasks, nil
}
