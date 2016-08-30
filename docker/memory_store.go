package docker

import (
	"sync"

	"github.com/docker/engine-api/types"
)

//StoreFilter defines a function to filter container
type StoreFilter func(*types.Container) bool

// ContainerStore keeps track of containers.
type ContainerStore struct {
	s map[string]*types.Container
	sync.RWMutex
}

// NewMemoryStore initializes a new memory store.
func NewMemoryStore() *ContainerStore {
	return &ContainerStore{
		s: make(map[string]*types.Container),
	}
}

// Add appends a new container to the memory store.
// It overrides the id if it existed before.
func (c *ContainerStore) Add(id string, cont *types.Container) {
	c.Lock()
	c.s[id] = cont
	c.Unlock()
}

// Get returns a container from the store by id.
func (c *ContainerStore) Get(id string) *types.Container {
	c.RLock()
	res := c.s[id]
	c.RUnlock()
	return res
}

// Delete removes a container from the store by id.
func (c *ContainerStore) Delete(id string) {
	c.Lock()
	delete(c.s, id)
	c.Unlock()
}

// List returns a list of containers from the store.
func (c *ContainerStore) List() []*types.Container {
	return c.all()
}

// Size returns the number of containers in the store.
func (c *ContainerStore) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.s)
}

// First returns the first container found in the store by the given filter.
func (c *ContainerStore) First(filter StoreFilter) *types.Container {
	for _, container := range c.s {
		if filter(container) {
			return container
		}
	}
	return nil
}

func (c *ContainerStore) all() []*types.Container {
	c.RLock()
	containers := make([]*types.Container, 0, len(c.s))
	for _, cont := range c.s {
		containers = append(containers, cont)
	}
	c.RUnlock()
	return containers
}
