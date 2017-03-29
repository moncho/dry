package appui

import (
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//AppUI holds initialized UI componets
type AppUI struct {
	DockerInfoWidget    *DockerInfo
	ContainerListWidget *DockerPs
	ImageListWidget     *DockerImagesRenderer
	DiskUsageWidget     *DockerDiskUsageRenderer
}

//NewAppUI creates AppUI
func NewAppUI(daemon docker.ContainerDaemon) *AppUI {
	di := NewDockerInfoBufferer(daemon)
	di.SetX(0)
	di.SetY(1)
	di.SetWidth(ui.ActiveScreen.Dimensions.Width)
	return &AppUI{
		di,
		NewDockerPsRenderer(),
		NewDockerImagesRenderer(),
		NewDockerDiskUsageRenderer(ui.ActiveScreen.Dimensions.Height),
	}
}
