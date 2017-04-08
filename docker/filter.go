package docker

import "strings"

//ContainerFilter defines a function to filter container
type ContainerFilter func(*Container) bool

//ContainerFilters is a holder of predefined ContainerFilter(s)
//The intentions is that something like 'ContainerFilters.ByName("name")'
//can be used to declare a filter.
var ContainerFilters ContainerFilter

//Unfiltered does not filter containers
func (c ContainerFilter) Unfiltered() ContainerFilter {
	return func(c *Container) bool { return true }
}

//ByName filters containers by name
func (c ContainerFilter) ByName(name string) ContainerFilter {
	return func(c *Container) bool {
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
	return func(c *Container) bool {
		return strings.Contains(c.ID, id)
	}
}

//ByRunningState filters containers by its running state
func (c ContainerFilter) ByRunningState(running bool) ContainerFilter {
	return func(c *Container) bool {
		return IsContainerRunning(c) == running
	}
}

//Running filters out container that are not running
func (c ContainerFilter) Running() ContainerFilter {
	return c.ByRunningState(true)
}

//NotRunning filters out container that are running
func (c ContainerFilter) NotRunning() ContainerFilter {
	return c.ByRunningState(false)
}

//Filter applies the given filter to the given slice of containers
func Filter(c []*Container, filter ContainerFilter) []*Container {
	var containers []*Container
	for _, cont := range c {
		if filter == nil || filter(cont) {
			containers = append(containers, cont)
		}
	}
	return containers
}
