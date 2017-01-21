package docker

import (
	"strings"

	"github.com/docker/docker/api/types"
)

//ContainerFilter defines a function to filter container
type ContainerFilter func(*types.Container) bool

//ContainerFilters is a holder of predefined ContainerFilter(s)
//The intentions is that something like 'ContainerFilters.ByName("name")'
//can be used to declare a filter.
var ContainerFilters ContainerFilter

//Unfiltered does not filter containers
func (c ContainerFilter) Unfiltered() ContainerFilter {
	return func(c *types.Container) bool { return true }
}

//ByName filters containers by name
func (c ContainerFilter) ByName(name string) ContainerFilter {
	return func(c *types.Container) bool {
		for _, containerName := range c.Names {

			if strings.Contains(containerName, name) {
				return true
			}
		}
		return false
	}
}

//ByID filters containers by ID
func (c ContainerFilter) ByID(id string) ContainerFilter {
	return func(c *types.Container) bool {
		return strings.Contains(c.ID, id)
	}
}
