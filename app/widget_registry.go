package app

import (
	"sync"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

//widgetRegistry holds two sets of widgets:
// * those registered in the the registry when it was created, that
//   can be reused. These are the individually named widgets found on
//   this struct.
// * a list of widgets to be rendered on the next rendering.
type widgetRegistry struct {
	ContainerList *appui.ContainersWidget
	ContainerMenu *appui.ContainerMenuWidget
	DiskUsage     *appui.DockerDiskUsageRenderer
	DockerInfo    *appui.DockerInfo
	ImageList     *appui.DockerImagesWidget
	Monitor       *appui.Monitor
	Networks      *appui.DockerNetworksWidget
	Nodes         *swarm.NodesWidget
	NodeTasks     *swarm.NodeTasksWidget
	ServiceTasks  *swarm.ServiceTasksWidget
	ServiceList   *swarm.ServicesWidget
	Stacks        *swarm.StacksWidget
	StackTasks    *swarm.StacksTasksWidget
	MessageBar    *ui.ExpiringMessageWidget
	activeWidgets map[string]termui.Widget
	sync.Mutex
}

//NewWidgetRegistry creates the WidgetCatalog
func newWidgetRegistry(daemon docker.ContainerDaemon) *widgetRegistry {
	di := appui.NewDockerInfo(daemon)
	di.SetX(0)
	di.SetY(1)
	di.SetWidth(ui.ActiveScreen.Dimensions.Width)
	w := widgetRegistry{
		DockerInfo:    di,
		ContainerList: appui.NewContainersWidget(daemon, appui.MainScreenHeaderSize),
		ContainerMenu: appui.NewContainerMenuWidget(daemon, appui.MainScreenHeaderSize),
		ImageList:     appui.NewDockerImagesWidget(daemon, appui.MainScreenHeaderSize),
		DiskUsage:     appui.NewDockerDiskUsageRenderer(ui.ActiveScreen.Dimensions.Height),
		Monitor:       appui.NewMonitor(daemon, appui.MainScreenHeaderSize),
		Networks:      appui.NewDockerNetworksWidget(daemon, appui.MainScreenHeaderSize),
		Nodes:         swarm.NewNodesWidget(daemon, appui.MainScreenHeaderSize),
		NodeTasks:     swarm.NewNodeTasksWidget(daemon, appui.MainScreenHeaderSize),
		ServiceTasks:  swarm.NewServiceTasksWidget(daemon, appui.MainScreenHeaderSize),
		ServiceList:   swarm.NewServicesWidget(daemon, appui.MainScreenHeaderSize),
		Stacks:        swarm.NewStacksWidget(daemon, appui.MainScreenHeaderSize),
		StackTasks:    swarm.NewStacksTasksWidget(daemon, appui.MainScreenHeaderSize),
		activeWidgets: make(map[string]termui.Widget),
		MessageBar:    ui.NewExpiringMessageWidget(0, ui.ActiveScreen.Dimensions.Width, appui.DryTheme),
	}

	return &w
}

func (wr *widgetRegistry) add(w termui.Widget) {
	wr.Lock()
	defer wr.Unlock()
	if err := w.Mount(); err == nil {
		wr.activeWidgets[w.Name()] = w
	}
}

func (wr *widgetRegistry) remove(w termui.Widget) {
	wr.Lock()
	defer wr.Unlock()
	if err := w.Unmount(); err == nil {
		delete(wr.activeWidgets, w.Name())
	}
}
