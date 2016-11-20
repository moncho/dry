package docker

import (
	"sync"

	"github.com/docker/engine-api/types"
)

// ContainerStore keeps track of containers.
type ContainerStore struct {
	s map[string]*types.Container
	c []*types.Container
	sync.RWMutex
}

// NewMemoryStore initializes a new memory store.
func NewMemoryStore() *ContainerStore {
	return &ContainerStore{
		s: make(map[string]*types.Container),
	}
}

// NewMemoryStoreWithContainers creates a new memory store from the given container slice.
func NewMemoryStoreWithContainers(containers []*types.Container) *ContainerStore {
	store := NewMemoryStore()
	for _, container := range containers {
		store.add(container)
	}
	return store
}

func (c *ContainerStore) add(cont *types.Container) {
	//If a container with the given ID exists already it is replaced
	if _, ok := c.s[cont.ID]; ok {
		for pos, container := range c.c {
			if container.ID == cont.ID {
				c.c = append(c.c[0:pos], c.c[pos:]...)
				break
			}
		}
	} else {
		c.c = append(c.c, cont)
	}
	c.s[cont.ID] = cont
}

// Add appends a new container to the memory store.
// It overrides the id if it existed before.
func (c *ContainerStore) Add(cont *types.Container) {
	c.Lock()
	c.add(cont)
	c.Unlock()
}

// At returns a container from the store by its position in the store
func (c *ContainerStore) At(pos int) *types.Container {
	if pos < 0 || pos >= len(c.c) {
		return nil
	}
	c.RLock()
	res := c.c[pos]
	c.RUnlock()
	return res
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
	return c.all(ContainerFilters.Unfiltered())
}

// Sort sorts the store
func (c *ContainerStore) Sort(mode SortMode) {
	c.RLock()
	defer c.RUnlock()
	SortContainers(c.c, mode)
}

// Size returns the number of containers in the store.
func (c *ContainerStore) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.c)
}

// Filter returns containers found in the store by the given filter.
func (c *ContainerStore) Filter(filter ContainerFilter) []*types.Container {
	return c.all(filter)
}

func (c *ContainerStore) all(filter ContainerFilter) []*types.Container {
	c.RLock()
	var containers []*types.Container
	for _, cont := range c.c {
		if filter(cont) {
			containers = append(containers, cont)
		}
	}
	c.RUnlock()
	return containers
}
