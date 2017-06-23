package appui

import "github.com/moncho/dry/docker"

type column struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
	mode  docker.SortMode
}

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
