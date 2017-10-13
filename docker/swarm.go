package docker

import (
	"context"
	"errors"
	"io"

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

//NodeChangeAvailability changes the availability of the given node
func (daemon *DockerDaemon) NodeChangeAvailability(nodeID string, availabilty swarm.NodeAvailability) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	node, _, err := daemon.client.NodeInspectWithRaw(ctx, nodeID)
	if err != nil {
		return err
	}

	node.Spec.Availability = availabilty
	err = daemon.client.NodeUpdate(ctx, nodeID, node.Version, node.Spec)
	if err == nil {
		return nil
	}
	return pkgError.Wrapf(err, "Error changing node %s availability", nodeID)
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

//NodeTasks returns the tasks being run by the given node
func (daemon *DockerDaemon) NodeTasks(nodeID string) ([]swarm.Task, error) {

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

//ResolveNode will attempt to resolve the given node ID to a name.
func (daemon *DockerDaemon) ResolveNode(id string) (string, error) {
	return daemon.resolve(swarm.Node{}, id)
}

//ResolveService will attempt to resolve the given service ID to a name.
func (daemon *DockerDaemon) ResolveService(id string) (string, error) {
	return daemon.resolve(swarm.Service{}, id)
}
func (daemon *DockerDaemon) resolve(t interface{}, id string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.resolver.Resolve(ctx, t, id)
}

//Service returns service details of the service with the given id
func (daemon *DockerDaemon) Service(id string) (*swarm.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	service, _, err := daemon.client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{InsertDefaults: true})
	if err == nil {
		return &service, nil
	}
	return nil, pkgError.Wrapf(err, "Error retrieving service with id %s", id)

}

//ServiceLogs returns logs of the service with the given id
func (daemon *DockerDaemon) ServiceLogs(id string) (io.ReadCloser, error) {

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Follow:     true,
		Details:    true,
	}
	return daemon.client.ServiceLogs(context.Background(), id, options)
}

//Services returns the services known by the Swarm
func (daemon *DockerDaemon) Services() ([]swarm.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.ServiceList(ctx, types.ServiceListOptions{})
}

//ServiceRemove removes the service with the given in
func (daemon *DockerDaemon) ServiceRemove(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.ServiceRemove(ctx, id)
}

//ServiceScale scales the given service by the given number of replicas
func (daemon *DockerDaemon) ServiceScale(id string, replicas uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	service, _, err := daemon.client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	serviceMode := &service.Spec.Mode
	if serviceMode.Replicated == nil {
		return errors.New("scale can only be used with replicated mode")
	}

	serviceMode.Replicated.Replicas = &replicas

	_, err = daemon.client.ServiceUpdate(
		ctx,
		id,
		service.Version,
		service.Spec,
		types.ServiceUpdateOptions{})
	return err

}

//ServiceTasks returns the tasks being run that belong to the given list of services
func (daemon *DockerDaemon) ServiceTasks(services ...string) ([]swarm.Task, error) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	filter := filters.NewArgs()
	for _, service := range services {
		filter.Add("service", service)
	}

	nodeTasks, err := daemon.client.TaskList(ctx, types.TaskListOptions{Filters: filter})

	if err == nil {
		return nodeTasks, nil
	}
	return nil, pkgError.Wrap(err, "Error retrieving task list")
}

//NewNodeAvailability builds NodeAvailability from the given string
func NewNodeAvailability(availability string) swarm.NodeAvailability {
	return swarm.NodeAvailability(availability)
}
