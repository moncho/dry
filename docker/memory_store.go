package docker

import (
	"sync"

	dockerAPI "github.com/docker/docker/client"
)

// ContainerStore defines a container storage.
type ContainerStore interface {
	Get(id string) *Container
	List() []*Container
	Remove(id string)
	Size() int
}

// inMemoryContainerStore is an in-memory container store backed up by a Docker daemon.
type inMemoryContainerStore struct {
	s      map[string]*Container
	c      []*Container
	client dockerAPI.ContainerAPIClient
	sync.RWMutex
}

// NewDockerContainerStore creates a new Docker container store that will use the given Docker
// daemon client to retrieve container information.
func NewDockerContainerStore(client dockerAPI.ContainerAPIClient) (ContainerStore, error) {
	containers, err := containers(client)
	if err != nil {
		return nil, err
	}
	store := &inMemoryContainerStore{
		s:      make(map[string]*Container),
		client: client,
	}
	for _, container := range containers {
		store.add(container)
	}
	return store, nil
}

func (c *inMemoryContainerStore) add(cont *Container) {
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
func (c *inMemoryContainerStore) Get(id string) *Container {
	c.RLock()
	res := c.s[id]
	c.RUnlock()
	return res
}

// List returns a list of containers from the store.
func (c *inMemoryContainerStore) List() []*Container {
	return c.all(nil)
}

// Remove removes a container from the store by id.
func (c *inMemoryContainerStore) Remove(id string) {
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

// Size returns the number of containers in the store.
func (c *inMemoryContainerStore) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.c)
}

// Filter returns containers found in the store by the given filter.
func (c *inMemoryContainerStore) Filter(filter ContainerFilter) []*Container {
	return c.all(filter)
}

func (c *inMemoryContainerStore) all(filter ContainerFilter) []*Container {
	c.RLock()
	defer c.RUnlock()

	var containers []*Container
	for _, cont := range c.c {
		if filter == nil || filter(cont) {
			containers = append(containers, cont)
		}
	}
	return containers
}
