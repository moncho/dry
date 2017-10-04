package appui

import "strings"

type containerRowFilter func(*ContainerRow) bool

var containerRowFilters containerRowFilter

//Unfiltered does not filter containers
func (cf containerRowFilter) Unfiltered() containerRowFilter {
	return func(c *ContainerRow) bool { return true }
}

//ByName filters containers by name
func (cf containerRowFilter) ByName(name string) containerRowFilter {
	return func(c *ContainerRow) bool {
		return strings.Contains(c.Names.Text, name)
	}
}

//ByID filters containers by ID
func (cf containerRowFilter) ByID(id string) containerRowFilter {
	return func(c *ContainerRow) bool {
		return strings.Contains(c.ID.Text, id)
	}
}

//ByRunningState filters containers by its running state
func (cf containerRowFilter) ByRunningState(running bool) containerRowFilter {
	return func(c *ContainerRow) bool {
		return c.running
	}
}

//Running filters out container that are not running
func (cf containerRowFilter) Running() containerRowFilter {
	return cf.ByRunningState(true)
}

//NotRunning filters out container that are running
func (cf containerRowFilter) NotRunning() containerRowFilter {
	return cf.ByRunningState(false)
}

//Apply applies this filter to the given slice of containers
func (cf containerRowFilter) Apply(c []*ContainerRow) []*ContainerRow {
	var containers []*ContainerRow
	for _, cont := range c {
		if cf(cont) {
			containers = append(containers, cont)
		}
	}
	return containers
}
