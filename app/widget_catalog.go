package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//WidgetCatalog holds initialized UI components that are ready
//to be rendered.
//For now, it holds named components.
type WidgetCatalog struct {
	DockerInfo    *appui.DockerInfo
	ContainerList *appui.ContainersWidget
	ImageList     *appui.DockerImagesRenderer
	DiskUsage     *appui.DockerDiskUsageRenderer
	NodeTasks     *swarm.NodeTasksWidget
}

//NewWidgetCatalog creates the WidgetCatalog
func NewWidgetCatalog(daemon docker.ContainerDaemon) *WidgetCatalog {
	di := appui.NewDockerInfo(daemon)
	di.SetX(0)
	di.SetY(1)
	di.SetWidth(ui.ActiveScreen.Dimensions.Width)
	return &WidgetCatalog{
		DockerInfo:    di,
		ContainerList: appui.NewContainersWidget(viewStartingLine),
		ImageList:     appui.NewDockerImagesRenderer(),
		DiskUsage:     appui.NewDockerDiskUsageRenderer(ui.ActiveScreen.Dimensions.Height),
	}
}
