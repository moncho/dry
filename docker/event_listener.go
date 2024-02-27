package docker

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types/events"
)

// SourceType is a representation of the sources types that might emit Docker events
type SourceType string

const (
	//ContainerSource for events emitted by Docker containers
	ContainerSource = SourceType("container")

	//DaemonSource for events emitted by the Docker daemon
	DaemonSource = SourceType("daemon")

	//ImageSource for events emitted by Docker images
	ImageSource = SourceType("image")

	//NetworkSource for events emitted by Docker networks
	NetworkSource = SourceType("network")

	//PluginSource for events emitted by Docker plugins
	PluginSource = SourceType("plugin")

	//VolumeSource for events emitted by Docker volumes
	VolumeSource = SourceType("volume")

	//ServiceSource for events emitted by Docker services
	ServiceSource = SourceType("service")

	//NodeSource for events emitted by Docker nodes
	NodeSource = SourceType("node")

	//SecretSource for events emitted by Docker secrets
	SecretSource = SourceType("secret")
)

// CallbackRegistry d
type CallbackRegistry interface {
	Register(actor SourceType, callback EventCallback)
}

// GlobalRegistry is a globally available CallbackRegistry
var GlobalRegistry CallbackRegistry

// callbackNotifier should be registered to receive events from Docker
var callbackNotifier EventCallback

func init() {
	r := &registry{actions: make(map[SourceType][]EventCallback)}
	GlobalRegistry = r
	callbackNotifier = notifyCallbacks(r)
}

type registry struct {
	actions map[SourceType][]EventCallback
	sync.RWMutex
}

// Register registers the interest of the given callback on messages from the given source
func (r *registry) Register(source SourceType, callback EventCallback) {
	r.Lock()
	defer r.Unlock()

	r.actions[source] = append(r.actions[source], callback)
}

func notifyCallbacks(r *registry) EventCallback {
	return func(ctx context.Context, message events.Message) error {
		r.RLock()
		defer r.RUnlock()
		actor := SourceType(message.Type)
		for _, c := range r.actions[actor] {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				go func(callback EventCallback) {
					callback(ctx, message)
				}(c)
			}
		}
		return nil
	}
}
