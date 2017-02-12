package appui

import "github.com/moncho/dry/docker"

//AppUI holds initialized UI componets
type AppUI struct {
	DockerInfo         string
	ContainerComponent *DockerPs
	DiskUsageComponet  *DockerDiskUsageRenderer
}

//NewAppUI creates AppUI
func NewAppUI(daemon docker.ContainerDaemon, screenHeight int) *AppUI {
	dockerInfo := dockerInfo(daemon)
	return &AppUI{
		dockerInfo,
		NewDockerPsRenderer(dockerInfo, screenHeight),
		NewDockerDiskUsageRenderer(dockerInfo, screenHeight),
	}
}
