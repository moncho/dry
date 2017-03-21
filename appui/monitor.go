package appui

import (
	"context"
	"time"

	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

//Monitor is a self-refreshing ui component that shows monitoring information about docker
//containers.
type Monitor struct {
	*termui.Grid
	screen         *ui.Screen
	containerCount int
	openChannels   []*docker.StatsChannel
}

//NewMonitor creates a new Monitor component that will render itself on the given screen
//at the given position and with the given width.
func NewMonitor(screen *ui.Screen, daemon docker.ContainerDaemon, y int) *Monitor {
	height := screen.Height - MainScreenHeaderSize - MainScreenFooterSize - 2
	g := termui.NewGrid(0, y, height, screen.Width)
	containers := daemon.ContainerStore().List()
	g.AddRows(DefaultMonitorTableHeader)
	var channels []*docker.StatsChannel
	for _, c := range containers {
		statsChan := daemon.OpenChannel(c)
		g.AddRows(NewContainerStatsRow(statsChan))
		channels = append(channels, statsChan)
	}
	g.Align()
	return &Monitor{g, screen, len(containers), channels}
}

//ContainerCount returns the number of containers known by this Monitor.
func (m *Monitor) ContainerCount() int {
	return m.containerCount
}

//RenderLoop makes this monitor to render itself until stopped.
func (m *Monitor) RenderLoop(ctx context.Context) {

	go func() {
		refreshTimer := time.NewTicker(500 * time.Millisecond)
		defer refreshTimer.Stop()
		defer func() {
			for _, c := range m.openChannels {
				c.Done <- struct{}{}
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case <-refreshTimer.C:
				m.screen.RenderBufferer(m)
				m.screen.Flush()
			}
		}
	}()

}
