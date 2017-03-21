package appui

import "github.com/moncho/dry/docker"

//AppUI holds initialized UI componets
type AppUI struct {
	DockerInfo         *DockerInfo
	ContainerComponent *DockerPs
	DiskUsageComponet  *DockerDiskUsageRenderer
}

//NewAppUI creates AppUI
func NewAppUI(daemon docker.ContainerDaemon, screenHeight, screenWidth int) *AppUI {
	di := NewDockerInfoBufferer(daemon)
	di.SetX(0)
	di.SetY(1)
	di.SetWidth(screenWidth)
	return &AppUI{
		di,
		NewDockerPsRenderer(screenHeight),
		NewDockerDiskUsageRenderer(screenHeight),
	}
}
