package app

import (
	"context"
	"image"
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
	screen := screen{ui.ActiveScreen}
	w := widgetRegistry{
		DockerInfo:    di,
		ContainerList: appui.NewContainersWidget(daemon, screen),
		ContainerMenu: appui.NewContainerMenuWidget(daemon, screen),
		ImageList:     appui.NewDockerImagesWidget(daemon.Images, screen),
		DiskUsage:     appui.NewDockerDiskUsageRenderer(height),
		Monitor:       appui.NewMonitor(daemon, screen),
		Networks:      appui.NewDockerNetworksWidget(daemon, screen),
		Nodes:         swarm.NewNodesWidget(daemon, screen),
		NodeTasks:     swarm.NewNodeTasksWidget(daemon, screen),
		ServiceTasks:  swarm.NewServiceTasksWidget(daemon, screen),
		ServiceList:   swarm.NewServicesWidget(daemon, screen),
		Stacks:        swarm.NewStacksWidget(daemon, screen),
		StackTasks:    swarm.NewStacksTasksWidget(daemon, screen),
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

type screen struct {
	*ui.Screen
}

func (s screen) Bounds() image.Rectangle {
	dim := s.Screen.Dimensions()
	y := appui.MainScreenHeaderSize
	return image.Rect(0, y, dim.Width, dim.Height-y)
}

func (s screen) Cursor() *ui.Cursor {
	return s.Screen.Cursor()
}
