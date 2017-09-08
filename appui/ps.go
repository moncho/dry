package appui

import "github.com/moncho/dry/docker"

//DockerPsRenderData holds information that might be
//used during ps rendering
type DockerPsRenderData struct {
	containers    []*docker.Container
	sortMode      docker.SortMode
	filterPattern string
}

//NewDockerPsRenderData creates render data structs
func NewDockerPsRenderData(containers []*docker.Container, sortMode docker.SortMode, filter string) *DockerPsRenderData {
	return &DockerPsRenderData{
		containers:    containers,
		sortMode:      sortMode,
		filterPattern: filter,
	}
}
