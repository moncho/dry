package docker

import (
	"sync"

	dockerAPI "github.com/docker/docker/client"
)

//ContainerStore defines a container storage.
type ContainerStore interface {
	Get(id string) *Container
	List() []*Container
	Remove(id string)
	Size() int
}

// InMemoryContainerStore is an in-memory container store backed up by a Docker daemon.
type InMemoryContainerStore struct {
	s      map[string]*Container
	c      []*Container
	client dockerAPI.ContainerAPIClient
	sync.RWMutex
}

//NewDockerContainerStore creates a new Docker container store that will use the given Docker
//daemon client to retrieve container information.
func NewDockerContainerStore(client dockerAPI.ContainerAPIClient) (ContainerStore, error) {
	containers, err := containers(client)
	if err != nil {
		return nil, err
	}
	store := &InMemoryContainerStore{
		s:      make(map[string]*Container),
		client: client,
	}
	for _, container := range containers {
		store.add(container)
	}
	return store, nil
}

func (c *InMemoryContainerStore) add(cont *Container) {
	c.Lock()
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
	c.Unlock()

}

// Get returns a container from the store by id.
func (c *InMemoryContainerStore) Get(id string) *Container {
	c.RLock()
	res := c.s[id]
	c.RUnlock()
	return res
}

// List returns a list of containers from the store.
func (c *InMemoryContainerStore) List() []*Container {
	return c.all(nil)
}

// Remove removes a container from the store by id.
func (c *InMemoryContainerStore) Remove(id string) {
	c.Lock()
	delete(c.s, id)
	for pos, container := range c.c {
		if container.ID == id {
			c.c = append(c.c[0:pos], c.c[pos+1:]...)
			break
		}
	}
	c.Unlock()
}

// Sort sorts the store
func (c *InMemoryContainerStore) Sort(mode SortMode) []*Container {
	c.RLock()
	defer c.RUnlock()
	containers := c.List()
	SortContainers(containers, mode)
	return containers
}

// Size returns the number of containers in the store.
func (c *InMemoryContainerStore) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.c)
}

// Filter returns containers found in the store by the given filter.
func (c *InMemoryContainerStore) Filter(filter ContainerFilter) []*Container {
	return c.all(filter)
}

func (c *InMemoryContainerStore) all(filter ContainerFilter) []*Container {
	c.RLock()
	var containers []*Container
	for _, cont := range c.c {
		if filter == nil || filter(cont) {
			containers = append(containers, cont)
		}
	}
	c.RUnlock()
	return containers
}
