package app

import "github.com/moncho/dry/docker"

// State is a naive attempt to represent application state
type State struct {
	changed              bool
	showingAllContainers bool
	Paused               bool
	viewMode             viewMode
	SortMode             docker.SortMode
}
