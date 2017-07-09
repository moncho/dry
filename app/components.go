package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//Widgets holds initialized UI componets
type Widgets struct {
	DockerInfo    *appui.DockerInfo
	ContainerList *appui.ContainersWidget
	ImageList     *appui.DockerImagesRenderer
	DiskUsage     *appui.DockerDiskUsageRenderer
	NodeTasks     *swarm.NodeTasksWidget
}

//NewAppUI creates AppUI
func NewAppUI(daemon docker.ContainerDaemon) *Widgets {
	di := appui.NewDockerInfo(daemon)
	di.SetX(0)
	di.SetY(1)
	di.SetWidth(ui.ActiveScreen.Dimensions.Width)
	return &Widgets{
		DockerInfo:    di,
		ContainerList: appui.NewContainersWidget(viewStartingLine),
		ImageList:     appui.NewDockerImagesRenderer(),
		DiskUsage:     appui.NewDockerDiskUsageRenderer(ui.ActiveScreen.Dimensions.Height),
	}
}
