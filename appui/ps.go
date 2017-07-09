package appui

import "github.com/moncho/dry/docker"

//DockerPsRenderData holds information that might be
//used during ps rendering
type DockerPsRenderData struct {
	containers []*docker.Container
	sortMode   docker.SortMode
}

//NewDockerPsRenderData creates render data structs
func NewDockerPsRenderData(containers []*docker.Container, sortMode docker.SortMode) *DockerPsRenderData {
	return &DockerPsRenderData{
		containers: containers,
		sortMode:   sortMode,
	}
}
