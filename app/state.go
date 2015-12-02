package app

import "github.com/moncho/dry/docker"

// AppState is a naive attempt to represent application state
type AppState struct {
	changed              bool
	message              string
	showingAllContainers bool
	Paused               bool
	ShowingHelp          bool
	viewMode             viewMode
	SortMode             docker.SortMode
}

func (appState AppState) Render() string {
	var state = "Help"
	switch appState.viewMode {
	case Main:
		{
			if appState.showingAllContainers {
				state = "Showing: <white>All Containers</>"
			} else {
				state = "Showing: <white>Running Containers</>"
			}
			if appState.message != "" {
				state += " <red>" + appState.message + "</>"
				//message is just shown once
				appState.message = ""
			}
		}
	default:
		{
			state = ""
		}
	}
	return state
}
