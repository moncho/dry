package docker

import (
	"context"
	"errors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// idResolver provides ID to Name resolution.
type idResolver struct {
	client    client.APIClient
	noResolve bool
	cache     map[string]string
}

// New creates a new IDResolver.
func newResolver(client client.APIClient, noResolve bool) *idResolver {
	return &idResolver{
		client:    client,
		noResolve: noResolve,
		cache:     make(map[string]string),
	}
}

func (r *idResolver) get(ctx context.Context, t interface{}, id string) (string, error) {
	switch t.(type) {
	case swarm.Node:
		node, _, err := r.client.NodeInspectWithRaw(ctx, id)
		if err != nil {
			return id, nil
		}
		if node.Spec.Annotations.Name != "" {
			return node.Spec.Annotations.Name, nil
		}
		if node.Description.Hostname != "" {
			return node.Description.Hostname, nil
		}
		return id, nil
	case swarm.Service:
		service, _, err := r.client.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
		if err != nil {
			return id, nil
		}
		return service.Spec.Annotations.Name, nil
	default:
		return "", errors.New("unsupported type")
	}

}

// Resolve will attempt to resolve an ID to a Name by querying the manager.
func (r *idResolver) Resolve(ctx context.Context, t interface{}, id string) (string, error) {
	if r.noResolve {
		return id, nil
	}
	if name, ok := r.cache[id]; ok {
		return name, nil
	}
	name, err := r.get(ctx, t, id)
	if err != nil {
		return "", err
	}
	r.cache[id] = name
	return name, nil
}
