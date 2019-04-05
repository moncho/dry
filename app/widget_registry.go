package app

import (
	"context"
	"sync"
	"time"

	"github.com/docker/docker/api/types/events"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

//widgetRegistry holds references to two types of widgets:
// * widgets that hold information that does not change or widgets
//   that hold information that is worth updating only when is changed.
//   These are all the widget tracked with a field in the struct.
// * a set of widgets to be rendered on the next rendering phase.
//
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

	sync.Mutex
	activeWidgets map[string]termui.Widget
}

//initRegistry creates a widget registry with its widget ready to be used
func initRegistry(daemon docker.ContainerDaemon) *widgetRegistry {
	di := appui.NewDockerInfo(daemon)
	di.SetX(0)
	di.SetY(1)
	di.SetWidth(ui.ActiveScreen.Dimensions.Width)
	w := widgetRegistry{
		DockerInfo:    di,
		ContainerList: appui.NewContainersWidget(daemon, appui.MainScreenHeaderSize),
		ContainerMenu: appui.NewContainerMenuWidget(daemon, appui.MainScreenHeaderSize),
		ImageList:     appui.NewDockerImagesWidget(daemon.Images, appui.MainScreenHeaderSize),
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

	refreshOnContainerEvent(w.ContainerList, daemon)
	refreshOnDockerEvent(docker.ImageSource, w.ImageList, Images)
	refreshOnDockerEvent(docker.NetworkSource, w.Networks, Networks)
	refreshOnDockerEvent(docker.NodeSource, w.Nodes, Nodes)
	refreshOnDockerEvent(docker.ServiceSource, w.ServiceList, Services)
	refreshOnDockerEvent(docker.ServiceSource, w.Stacks, Stacks)

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

var timeBetweenRefresh = 1000 * time.Millisecond

func refreshOnDockerEvent(source docker.SourceType, w termui.Widget, view viewMode) {
	last := time.Now()
	docker.GlobalRegistry.Register(
		source,
		func(ctx context.Context, m events.Message) error {
			if time.Since(last) > timeBetweenRefresh {
				last = time.Now()
				err := w.Unmount()
				if err != nil {
					return err
				}
				return refreshIfView(view)
			}
			return nil
		})
}
func refreshOnContainerEvent(w termui.Widget, daemon docker.ContainerDaemon) {
	last := time.Now()
	docker.GlobalRegistry.Register(
		docker.ContainerSource,
		func(ctx context.Context, m events.Message) error {
			if time.Since(last) > timeBetweenRefresh {
				last = time.Now()
				daemon.Refresh(func(e error) {
					err := w.Unmount()
					if err != nil {
						return
					}

					refreshIfView(Main)
				})
			}
			return nil
		})
}
