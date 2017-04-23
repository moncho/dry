package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//Widgets holds initialized UI componets
type Widgets struct {
	DockerInfoWidget    *appui.DockerInfo
	ContainerListWidget *appui.DockerPs
	ImageListWidget     *appui.DockerImagesRenderer
	DiskUsageWidget     *appui.DockerDiskUsageRenderer
	NodeTasksWidget     *swarm.NodeTasksWidget
}

//NewAppUI creates AppUI
func NewAppUI(daemon docker.ContainerDaemon) *Widgets {
	di := appui.NewDockerInfo(daemon)
	di.SetX(0)
	di.SetY(1)
	di.SetWidth(ui.ActiveScreen.Dimensions.Width)
	return &Widgets{
		DockerInfoWidget:    di,
		ContainerListWidget: appui.NewDockerPsRenderer(),
		ImageListWidget:     appui.NewDockerImagesRenderer(),
		DiskUsageWidget:     appui.NewDockerDiskUsageRenderer(ui.ActiveScreen.Dimensions.Height),
	}
}
