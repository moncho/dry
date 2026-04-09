package docker

import (
	"context"
	"errors"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
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
		res, err := r.client.NodeInspect(ctx, id, client.NodeInspectOptions{})
		if err != nil {
			return id, nil
		}
		if n := res.Node.Spec.Name; n != "" {
			return n, nil
		}
		if hn := res.Node.Description.Hostname; hn != "" {
			return hn, nil
		}
		return id, nil
	case swarm.Service:
		res, err := r.client.ServiceInspect(ctx, id, client.ServiceInspectOptions{})
		if err != nil {
			return id, nil
		}
		return res.Service.Spec.Name, nil
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
