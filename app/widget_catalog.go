package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

//WidgetRegistry holds two sets of widgets:
// * those registered in the the registry when it was created, that
//   can be reused. These are the individually named widgets found on
//   this struct.
// * a list of widgets to be rendered on the next rendering.
type WidgetRegistry struct {
	DockerInfo    *appui.DockerInfo
	ContainerList *appui.ContainersWidget
	ImageList     *appui.DockerImagesRenderer
	DiskUsage     *appui.DockerDiskUsageRenderer
	NodeTasks     *swarm.NodeTasksWidget
	ActiveWidgets []termui.Widget
}

//NewWidgetRegistry creates the WidgetCatalog
func NewWidgetRegistry(daemon docker.ContainerDaemon) *WidgetRegistry {
	di := appui.NewDockerInfo(daemon)
	di.SetX(0)
	di.SetY(1)
	di.SetWidth(ui.ActiveScreen.Dimensions.Width)
	return &WidgetRegistry{
		DockerInfo:    di,
		ContainerList: appui.NewContainersWidget(viewStartingLine),
		ImageList:     appui.NewDockerImagesRenderer(),
		DiskUsage:     appui.NewDockerDiskUsageRenderer(ui.ActiveScreen.Dimensions.Height),
	}
}
