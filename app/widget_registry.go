package app

import (
	"sync"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

// widgetRegistry holds references to two types of widgets:
//   - widgets that hold information that does not change or widgets
//     that hold information that is worth updating only when is changed.
//     These are all the widget tracked with a field in the struct.
//   - a set of widgets to be rendered on the next rendering phase.
type widgetRegistry struct {
	ContainerList *appui.ContainersWidget
	ContainerMenu *appui.ContainerMenuWidget
	DiskUsage     *appui.DockerDiskUsageRenderer
	DockerInfo    *appui.DockerInfo
	ImageList     *appui.DockerImagesWidget
	MessageBar    *ui.ExpiringMessageWidget
	Monitor       *appui.Monitor
	Networks      *appui.DockerNetworksWidget
	Nodes         *swarm.NodesWidget
	NodeTasks     *swarm.NodeTasksWidget
	ServiceTasks  *swarm.ServiceTasksWidget
	ServiceList   *swarm.ServicesWidget
	Stacks        *swarm.StacksWidget
	StackTasks    *swarm.StacksTasksWidget
	Volumes       *appui.VolumesWidget
	sync.RWMutex
	widgets map[string]termui.Widget
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
