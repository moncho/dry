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

const (
	// LabelNamespace is the label used to track stack resources
	//Copied from https://github.com/docker/cli/blob/master/cli/compose/convert/compose.go
	LabelNamespace = "com.docker.stack.namespace"
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
func (daemon *DockerDaemon) NodeChangeAvailability(nodeID string, availability swarm.NodeAvailability) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	node, _, err := daemon.client.NodeInspectWithRaw(ctx, nodeID)
	if err != nil {
		return err
	}

	node.Spec.Availability = availability
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
func (daemon *DockerDaemon) ServiceLogs(id string, since string) (io.ReadCloser, error) {

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Follow:     true,
		Details:    true,
		Since:      since,
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

//ServiceUpdate forces an update of the given service
func (daemon *DockerDaemon) ServiceUpdate(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	service, _, err := daemon.client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	service.Spec.TaskTemplate.ForceUpdate++

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

//Stacks returns the stack list
func (daemon *DockerDaemon) Stacks() ([]Stack, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	services, err := daemon.client.ServiceList(
		ctx,
		types.ServiceListOptions{Filters: getAllStacksFilter()})
	if err != nil {
		return nil, err
	}

	m := make(map[string]*Stack)
	for _, service := range services {
		labels := service.Spec.Labels
		name, ok := labels[LabelNamespace]
		if !ok {
			return nil, pkgError.Errorf("cannot get label %s for service %s",
				LabelNamespace, service.ID)
		}
		ztack, ok := m[name]
		if !ok {
			cc, err := daemon.StackConfigs(name)
			if err != nil {
				return nil, pkgError.Errorf("cannot get configs for stack %s",
					name)
			}
			nn, err := daemon.StackNetworks(name)
			if err != nil {
				return nil, pkgError.Errorf("cannot get networks for stack %s",
					name)
			}
			ss, err := daemon.StackSecrets(name)
			if err != nil {
				return nil, pkgError.Errorf("cannot get secrets for stack %s",
					name)
			}

			m[name] = &Stack{
				Name:     name,
				Services: 1,
				Configs:  len(cc),
				Secrets:  len(ss),
				Networks: len(nn),
			}

		} else {
			ztack.Services++
		}
	}
	var stacks []Stack
	for _, stack := range m {
		stacks = append(stacks, *stack)
	}
	return stacks, nil
}

//StackConfigs returns the configs created for the given stack
func (daemon *DockerDaemon) StackConfigs(stack string) ([]swarm.Config, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.ConfigList(
		ctx,
		types.ConfigListOptions{Filters: buildStackFilter(stack)})
}

//StackNetworks returns the networks created for the given stack
func (daemon *DockerDaemon) StackNetworks(stack string) ([]types.NetworkResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.NetworkList(
		ctx,
		types.NetworkListOptions{Filters: buildStackFilter(stack)})
}

//StackSecrets return the secrets created for the given stack
func (daemon *DockerDaemon) StackSecrets(stack string) ([]swarm.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.SecretList(
		ctx,
		types.SecretListOptions{Filters: buildStackFilter(stack)})
}

//StackServices returns the given stack service list
func (daemon *DockerDaemon) StackServices(stack string) ([]swarm.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	filter := buildStackFilter(stack)

	stackServices, err := daemon.client.ServiceList(ctx, types.ServiceListOptions{Filters: filter})

	if err == nil {
		return stackServices, nil
	}
	return nil, pkgError.Wrap(err, "Error retrieving stack service list")
}

//StackTasks returns the given stack task list
func (daemon *DockerDaemon) StackTasks(stack string) ([]swarm.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	filter := buildStackFilter(stack)

	stackTasks, err := daemon.client.TaskList(ctx, types.TaskListOptions{Filters: filter})

	if err == nil {
		return stackTasks, nil
	}
	return nil, pkgError.Wrap(err, "Error retrieving task list")
}

//Task returns the task with the given id
func (daemon *DockerDaemon) Task(id string) (swarm.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	task, _, err := daemon.client.TaskInspectWithRaw(ctx, id)

	if err == nil {
		return task, nil
	}
	return swarm.Task{}, pkgError.Wrapf(err, "Error retrieving task with ID: %s", id)

}
func buildStackFilter(stack string) filters.Args {
	filter := filters.NewArgs()
	filter.Add("label", "com.docker.stack.namespace="+stack)
	return filter
}
func getAllStacksFilter() filters.Args {
	filter := filters.NewArgs()
	filter.Add("label", LabelNamespace)
	return filter
}

//NewNodeAvailability builds NodeAvailability from the given string
func NewNodeAvailability(availability string) swarm.NodeAvailability {
	return swarm.NodeAvailability(availability)
}
