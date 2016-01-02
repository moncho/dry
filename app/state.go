package app

import "github.com/moncho/dry/docker"

// State is a naive attempt to represent application state
type State struct {
	changed              bool
	showingAllContainers bool
	Paused               bool
	ShowingHelp          bool
	viewMode             viewMode
	SortMode             docker.SortMode
}

//Render the state
func (appState State) Render() string {
	var state = "Help"
	switch appState.viewMode {
	case Main:
		{
			if appState.showingAllContainers {
				state = "Showing: <white>All Containers</>"
			} else {
				state = "Showing: <white>Running Containers</>"
			}
		}
	default:
		{
			state = ""
		}
	}
	return state
}
