package docker

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types/events"
)

//SourceType is a representation of the sources types that might emit Docker events
type SourceType string

//ContainerSource for events emitted by Docker containers
var ContainerSource = SourceType("container")

//DaemonSource for events emitted by the Docker daemon
var DaemonSource = SourceType("daemon")

//ImageSource for events emitted by Docker images
var ImageSource = SourceType("image")

//NetworkSource for events emitted by Docker networks
var NetworkSource = SourceType("network")

//PluginSource for events emitted by Docker plugins
var PluginSource = SourceType("plugin")

//VolumeSource for events emitted by Docker volumes
var VolumeSource = SourceType("volume")

//ServiceSource for events emitted by Docker services
var ServiceSource = SourceType("service")

//NodeSource for events emitted by Docker nodes
var NodeSource = SourceType("node")

//SecretSource for events emitted by Docker secrets
var SecretSource = SourceType("secret")

//CallbackRegistry d
type CallbackRegistry interface {
	Register(actor SourceType, callback EventCallback)
}

//GlobalRegistry is a globally available CallbackRegistry
var GlobalRegistry CallbackRegistry

func init() {
	GlobalRegistry = &registry{actions: make(map[SourceType][]EventCallback)}
}

type registry struct {
	actions map[SourceType][]EventCallback
	sync.Mutex
}

//Register registers the interest of the given callback on messages from the given source
func (r *registry) Register(source SourceType, callback EventCallback) {
	r.Lock()
	defer r.Unlock()

	r.actions[source] = append(r.actions[source], callback)
}

func notifyCallbacks(r *registry) EventCallback {
	return func(ctx context.Context, message events.Message) error {
		actor := SourceType(message.Type)
		for _, callback := range r.actions[actor] {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			go callback(ctx, message)
		}
		return nil
	}
}
