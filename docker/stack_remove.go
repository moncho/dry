package docker

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
)

// StackRemove removes the stack with the given in
func (daemon *DockerDaemon) StackRemove(stack string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	var errs []string
	services, err := daemon.StackServices(stack)
	if err != nil {
		return err
	}

	networks, err := daemon.StackNetworks(stack)
	if err != nil {
		return err
	}

	var secrets []swarm.Secret
	if versions.GreaterThanOrEqualTo(daemon.client.ClientVersion(), "1.25") {
		secrets, err = daemon.StackSecrets(stack)
		if err != nil {
			return err
		}
	}

	var configs []swarm.Config
	if versions.GreaterThanOrEqualTo(daemon.client.ClientVersion(), "1.30") {
		configs, err = daemon.StackConfigs(stack)
		if err != nil {
			return err
		}
	}

	if len(services)+len(networks)+len(secrets)+len(configs) == 0 {
		return fmt.Errorf("Nothing found in stack: %s", stack)
	}

	hasError := removeServices(ctx, daemon, services)
	hasError = removeSecrets(ctx, daemon, secrets) || hasError
	hasError = removeConfigs(ctx, daemon, configs) || hasError
	hasError = removeNetworks(ctx, daemon, networks) || hasError

	if hasError {
		errs = append(errs, fmt.Sprintf("Failed to remove some resources from stack: %s", stack))
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func sortServiceByName(services []swarm.Service) func(i, j int) bool {
	return func(i, j int) bool {
		return services[i].Spec.Name < services[j].Spec.Name
	}
}

func removeServices(
	ctx context.Context,
	daemon *DockerDaemon,
	services []swarm.Service) bool {
	var hasError bool
	sort.Slice(services, sortServiceByName(services))
	for _, service := range services {
		if err := daemon.client.ServiceRemove(ctx, service.ID); err != nil {
			hasError = true
		}
	}
	return hasError
}

func removeNetworks(
	ctx context.Context,
	daemon *DockerDaemon,
	networks []types.NetworkResource) bool {
	var hasError bool
	for _, network := range networks {
		if err := daemon.client.NetworkRemove(ctx, network.ID); err != nil {
			hasError = true
		}
	}
	return hasError
}

func removeSecrets(
	ctx context.Context,
	daemon *DockerDaemon,
	secrets []swarm.Secret) bool {
	var hasError bool
	for _, secret := range secrets {
		if err := daemon.client.SecretRemove(ctx, secret.ID); err != nil {
			hasError = true
		}
	}
	return hasError
}

func removeConfigs(
	ctx context.Context,
	daemon *DockerDaemon,
	configs []swarm.Config) bool {
	var hasError bool
	for _, config := range configs {
		if err := daemon.client.ConfigRemove(ctx, config.ID); err != nil {
			hasError = true
		}
	}
	return hasError
}
