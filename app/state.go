package app

import "github.com/moncho/dry/docker"

// AppState is a naive attempt to describe application state
type AppState struct {
	changed              bool
	message              string
	showingAllContainers bool
	Paused               bool
	ShowingHelp          bool
	showingStats         bool
	SortMode             docker.SortMode
}

func (appState *AppState) Render() string {
	if appState.showingStats {
		appState.showingStats = false
		return ""
	}
	var state = "Help"
	if !appState.ShowingHelp {
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
	return state
}
