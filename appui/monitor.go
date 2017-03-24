package appui

import (
	"context"
	"time"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//Monitor is a self-refreshing ui component that shows monitoring information about docker
//containers.
type Monitor struct {
	header        *monitorTableHeader
	rows          []*ContainerStatsRow
	openChannels  []*docker.StatsChannel
	x, y          int
	height, width int
}

//NewMonitor creates a new Monitor component that will render itself on the given screen
//at the given position and with the given width.
func NewMonitor(daemon docker.ContainerDaemon, y int) *Monitor {
	height := ui.ActiveScreen.Dimensions.Height - MainScreenHeaderSize - MainScreenFooterSize - 2
	containers := daemon.ContainerStore().Filter(docker.ContainerFilters.ByRunningState(true))
	var rows []*ContainerStatsRow
	var channels []*docker.StatsChannel
	for _, c := range containers {
		statsChan := daemon.OpenChannel(c)
		rows = append(rows, NewContainerStatsRow(statsChan))
		channels = append(channels, statsChan)
	}
	m := &Monitor{DefaultMonitorTableHeader, rows, channels, 0, y, height, ui.ActiveScreen.Dimensions.Width}
	m.align()
	return m
}

//ContainerCount returns the number of containers known by this Monitor.
func (m *Monitor) ContainerCount() int {
	return len(m.rows)
}

//Align aligns rows
func (m *Monitor) align() {
	y := m.y
	x := m.x
	width := m.width

	m.header.SetWidth(width)
	m.header.SetY(y)
	m.header.SetX(x)
	y += m.header.GetHeight()

	for _, r := range m.rows {
		r.SetY(y)
		r.SetX(x)
		y += r.GetHeight()
		r.SetWidth(width)
	}
}

//Buffer returns the content of this monitor as a termui.Buffer
func (m *Monitor) Buffer() gizaktermui.Buffer {
	buf := gizaktermui.NewBuffer()
	buf.Merge(DefaultMonitorTableHeader.Buffer())
	for _, r := range m.rows {
		buf.Merge(r.Buffer())
	}
	return buf
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
				ui.ActiveScreen.RenderBufferer(m)
				ui.ActiveScreen.Flush()
			}
		}
	}()

}
