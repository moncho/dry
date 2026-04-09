package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

const (
	// LabelNamespace is the label used to track stack resources
	// Copied from https://github.com/docker/cli/blob/master/cli/compose/convert/compose.go
	LabelNamespace = "com.docker.stack.namespace"
)

// Node returns the node with the given id
func (daemon *DockerDaemon) Node(id string) (*swarm.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.NodeInspect(ctx, id, client.NodeInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("retrieve node with id %s: %w", id, err)
	}
	return &res.Node, nil
}

// NodeChangeAvailability changes the availability of the given node
func (daemon *DockerDaemon) NodeChangeAvailability(nodeID string, availability swarm.NodeAvailability) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.NodeInspect(ctx, nodeID, client.NodeInspectOptions{})
	if err != nil {
		return err
	}

	res.Node.Spec.Availability = availability
	_, err = daemon.client.NodeUpdate(ctx, nodeID, client.NodeUpdateOptions{
		Version: res.Node.Version,
		Spec:    res.Node.Spec,
	})
	if err != nil {
		return fmt.Errorf("change node %s availability: %w", nodeID, err)
	}
	return nil
}

// Nodes returns the nodes that are part of the Swarm
func (daemon *DockerDaemon) Nodes() ([]swarm.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.NodeList(ctx, client.NodeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("retrieve node list: %w", err)
	}
	return res.Items, nil
}

// NodeTasks returns the tasks being run by the given node
func (daemon *DockerDaemon) NodeTasks(nodeID string) ([]swarm.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.TaskList(ctx, client.TaskListOptions{
		Filters: make(client.Filters).Add("node", nodeID),
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve tasks for node %s: %w", nodeID, err)
	}
	return res.Items, nil
}

// ResolveNode will attempt to resolve the given node ID to a name.
func (daemon *DockerDaemon) ResolveNode(id string) (string, error) {
	return daemon.resolve(swarm.Node{}, id)
}

// ResolveService will attempt to resolve the given service ID to a name.
func (daemon *DockerDaemon) ResolveService(id string) (string, error) {
	return daemon.resolve(swarm.Service{}, id)
}

func (daemon *DockerDaemon) resolve(t interface{}, id string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.resolver.Resolve(ctx, t, id)
}

// Service returns service details of the service with the given id
func (daemon *DockerDaemon) Service(id string) (*swarm.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.ServiceInspect(ctx, id, client.ServiceInspectOptions{InsertDefaults: true})
	if err != nil {
		return nil, fmt.Errorf("retrieve service with id %s: %w", id, err)
	}
	return &res.Service, nil
}

// ServiceLogs returns logs of the service with the given id
func (daemon *DockerDaemon) ServiceLogs(id string, since string, withTimestamps bool) (io.ReadCloser, error) {
	options := client.ServiceLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: withTimestamps,
		Follow:     true,
		Details:    true,
		Since:      since,
	}
	if strings.HasPrefix(since, "tail:") {
		options.Since = ""
		options.Tail = strings.TrimPrefix(since, "tail:")
		if _, err := strconv.Atoi(options.Tail); err != nil {
			return nil, fmt.Errorf("invalid service log tail value %q: %w", options.Tail, err)
		}
	}
	return daemon.client.ServiceLogs(context.Background(), id, options)
}

// Services returns the services known by the Swarm
func (daemon *DockerDaemon) Services() ([]swarm.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// ServiceRemove removes the service with the given in
func (daemon *DockerDaemon) ServiceRemove(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	_, err := daemon.client.ServiceRemove(ctx, id, client.ServiceRemoveOptions{})
	return err
}

// ServiceScale scales the given service by the given number of replicas
func (daemon *DockerDaemon) ServiceScale(id string, replicas uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	res, err := daemon.client.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	serviceMode := &res.Service.Spec.Mode
	if serviceMode.Replicated == nil {
		return errors.New("scale can only be used with replicated mode")
	}

	serviceMode.Replicated.Replicas = &replicas

	_, err = daemon.client.ServiceUpdate(ctx, id, client.ServiceUpdateOptions{
		Version: res.Service.Version,
		Spec:    res.Service.Spec,
	})
	return err
}

// ServiceUpdate forces an update of the given service
func (daemon *DockerDaemon) ServiceUpdate(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	res, err := daemon.client.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	res.Service.Spec.TaskTemplate.ForceUpdate++

	_, err = daemon.client.ServiceUpdate(ctx, id, client.ServiceUpdateOptions{
		Version: res.Service.Version,
		Spec:    res.Service.Spec,
	})
	return err
}

// ServiceTasks returns the tasks being run that belong to the given list of services
func (daemon *DockerDaemon) ServiceTasks(services ...string) ([]swarm.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.TaskList(ctx, client.TaskListOptions{
		Filters: make(client.Filters).Add("service", services...),
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve task list: %w", err)
	}
	return res.Items, nil
}

// Stacks returns the stack list
func (daemon *DockerDaemon) Stacks() ([]Stack, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	res, err := daemon.client.ServiceList(ctx, client.ServiceListOptions{
		Filters: getAllStacksFilter(),
	})
	if err != nil {
		return nil, err
	}

	m := make(map[string]*Stack)
	for _, service := range res.Items {
		labels := service.Spec.Labels
		name, ok := labels[LabelNamespace]
		if !ok {
			return nil, fmt.Errorf("no label %s for service %s", LabelNamespace, service.ID)
		}
		ztack, ok := m[name]
		if !ok {
			cc, err := daemon.StackConfigs(name)
			if err != nil {
				return nil, fmt.Errorf("get configs for stack %s", name)
			}
			nn, err := daemon.StackNetworks(name)
			if err != nil {
				return nil, fmt.Errorf("get networks for stack %s", name)
			}
			ss, err := daemon.StackSecrets(name)
			if err != nil {
				return nil, fmt.Errorf("get secrets for stack %s", name)
			}

			m[name] = &Stack{
				Name:         name,
				Services:     1,
				Orchestrator: "Swarm",
				Configs:      len(cc),
				Secrets:      len(ss),
				Networks:     len(nn),
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

// StackConfigs returns the configs created for the given stack
func (daemon *DockerDaemon) StackConfigs(stack string) ([]swarm.Config, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.ConfigList(ctx, client.ConfigListOptions{
		Filters: make(client.Filters).Add("stack", stack),
	})
	if err != nil {
		return nil, err
	}
	return res.Items, err
}

// StackNetworks returns the networks created for the given stack
func (daemon *DockerDaemon) StackNetworks(stack string) ([]network.Inspect, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.NetworkList(ctx, client.NetworkListOptions{
		Filters: buildStackFilter(stack),
	})
	if err != nil {
		return nil, err
	}
	out := make([]network.Inspect, 0, len(res.Items))
	for _, nw := range res.Items {
		out = append(out, network.Inspect{
			Network: nw.Network,
		})
	}
	return out, nil
}

// StackSecrets return the secrets created for the given stack
func (daemon *DockerDaemon) StackSecrets(stack string) ([]swarm.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.SecretList(ctx, client.SecretListOptions{
		Filters: buildStackFilter(stack),
	})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// StackServices returns the given stack service list
func (daemon *DockerDaemon) StackServices(stack string) ([]swarm.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.ServiceList(ctx, client.ServiceListOptions{
		Filters: buildStackFilter(stack),
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve stack service list: %w", err)
	}
	return res.Items, nil
}

// StackTasks returns the given stack task list
func (daemon *DockerDaemon) StackTasks(stack string) ([]swarm.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	res, err := daemon.client.TaskList(ctx, client.TaskListOptions{
		Filters: buildStackFilter(stack),
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve stack task list: %w", err)
	}
	return res.Items, nil
}

// Task returns the task with the given id
func (daemon *DockerDaemon) Task(id string) (swarm.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	res, err := daemon.client.TaskInspect(ctx, id, client.TaskInspectOptions{})
	if err != nil {
		return swarm.Task{}, fmt.Errorf("retrieve task with id %s: %w", id, err)
	}
	return res.Task, nil
}

func buildStackFilter(stack string) client.Filters {
	return make(client.Filters).Add("label", "com.docker.stack.namespace="+stack)
}

func getAllStacksFilter() client.Filters {
	return make(client.Filters).Add("label", LabelNamespace)
}

// NewNodeAvailability builds NodeAvailability from the given string
func NewNodeAvailability(availability string) swarm.NodeAvailability {
	return swarm.NodeAvailability(availability)
}
