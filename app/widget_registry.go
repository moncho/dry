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

	sync.RWMutex
	widgets map[string]termui.Widget
}

//initRegistry creates a widget registry with its widget ready to be used
func initRegistry(daemon docker.ContainerDaemon) *widgetRegistry {
	d := ui.ActiveScreen.Dimensions()
	height, width := d.Height, d.Width
	di := appui.NewDockerInfo(daemon)
	di.SetX(0)
	di.SetY(1)
	di.SetWidth(width)
	w := widgetRegistry{
		DockerInfo:    di,
		ContainerList: appui.NewContainersWidget(daemon, ui.ActiveScreen, appui.MainScreenHeaderSize),
		ContainerMenu: appui.NewContainerMenuWidget(daemon, ui.ActiveScreen, appui.MainScreenHeaderSize),
		ImageList:     appui.NewDockerImagesWidget(daemon.Images, ui.ActiveScreen, appui.MainScreenHeaderSize),
		DiskUsage:     appui.NewDockerDiskUsageRenderer(height),
		Monitor:       appui.NewMonitor(daemon, ui.ActiveScreen, appui.MainScreenHeaderSize),
		Networks:      appui.NewDockerNetworksWidget(daemon, ui.ActiveScreen, appui.MainScreenHeaderSize),
		Nodes:         swarm.NewNodesWidget(daemon, ui.ActiveScreen, appui.MainScreenHeaderSize),
		NodeTasks:     swarm.NewNodeTasksWidget(daemon, ui.ActiveScreen, appui.MainScreenHeaderSize),
		ServiceTasks:  swarm.NewServiceTasksWidget(daemon, ui.ActiveScreen, appui.MainScreenHeaderSize),
		ServiceList:   swarm.NewServicesWidget(daemon, ui.ActiveScreen, appui.MainScreenHeaderSize),
		Stacks:        swarm.NewStacksWidget(daemon, ui.ActiveScreen, appui.MainScreenHeaderSize),
		StackTasks:    swarm.NewStacksTasksWidget(daemon, ui.ActiveScreen, appui.MainScreenHeaderSize),
		widgets:       make(map[string]termui.Widget),
		MessageBar:    ui.NewExpiringMessageWidget(0, ui.ActiveScreen),
	}

	refreshOnContainerEvent(w.ContainerList, daemon)
	refreshOnDockerEvent(docker.ImageSource, w.ImageList, Images)
	refreshOnDockerEvent(docker.NetworkSource, w.Networks, Networks)
	refreshOnDockerEvent(docker.NodeSource, w.Nodes, Nodes)
	refreshOnDockerEvent(docker.ServiceSource, w.ServiceList, Services)
	refreshOnDockerEvent(docker.ServiceSource, w.Stacks, Stacks)

	return &w
}

func (wr *widgetRegistry) add(w termui.Widget) error {
	wr.Lock()
	defer wr.Unlock()
	err := w.Mount()
	if err == nil {
		wr.widgets[w.Name()] = w
	}
	return err
}

func (wr *widgetRegistry) remove(w termui.Widget) error {
	wr.Lock()
	defer wr.Unlock()
	delete(wr.widgets, w.Name())
	return w.Unmount()
}

func (wr *widgetRegistry) activeWidgets() []termui.Widget {
	wr.RLock()
	defer wr.RUnlock()
	widgets := make([]termui.Widget, len(wr.widgets))
	i := 0
	for _, widget := range wr.widgets {
		widgets[i] = widget
		i++
	}
	return widgets
}
func (wr *widgetRegistry) reload() {

}

var timeBetweenRefresh = 250 * time.Millisecond

func refreshOnDockerEvent(source docker.SourceType, w termui.Widget, view viewMode) {
	last := time.Now()
	var lock sync.Mutex
	docker.GlobalRegistry.Register(
		source,
		func(ctx context.Context, m events.Message) error {
			lock.Lock()
			defer lock.Unlock()
			if time.Since(last) < timeBetweenRefresh {
				return nil
			}
			last = time.Now()
			err := w.Unmount()
			if err != nil {
				return err
			}
			return refreshIfView(view)
		})
}
func refreshOnContainerEvent(w termui.Widget, daemon docker.ContainerDaemon) {
	last := time.Now()
	var lock sync.Mutex
	docker.GlobalRegistry.Register(
		docker.ContainerSource,
		func(ctx context.Context, m events.Message) error {
			lock.Lock()
			defer lock.Unlock()
			if time.Since(last) < timeBetweenRefresh {
				return nil
			}
			last = time.Now()
			daemon.Refresh(func(e error) {
				err := w.Unmount()
				if err != nil {
					return
				}

				refreshIfView(Main)
			})
			return nil
		})
}
