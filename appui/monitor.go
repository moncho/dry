package appui

import (
	"context"
	"sync"
	"time"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//Monitor is a self-refreshing ui component that shows monitoring information about docker
//containers.
type Monitor struct {
	header               *MonitorTableHeader
	daemon               docker.ContainerDaemon
	rows                 []*ContainerStatsRow
	openChannels         []*docker.StatsChannel
	selectedIndex        int
	offset               int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewMonitor creates a new Monitor component that will render itself on the given screen
//at the given position and with the given width.
func NewMonitor(daemon docker.ContainerDaemon, y int) *Monitor {
	height := MainScreenAvailableHeight()
	m := Monitor{
		header:        defaultMonitorTableHeader,
		daemon:        daemon,
		selectedIndex: 0,
		offset:        0,
		x:             0,
		y:             y,
		height:        height,
		width:         ui.ActiveScreen.Dimensions.Width}
	return &m
}

//Buffer returns the content of this monitor as a termui.Buffer
func (m *Monitor) Buffer() gizaktermui.Buffer {
	m.Lock()
	defer m.Unlock()
	y := m.y
	buf := gizaktermui.NewBuffer()

	widgetHeader := WidgetHeader("Containers", m.RowCount(), "")
	widgetHeader.Y = y
	buf.Merge(widgetHeader.Buffer())
	y += widgetHeader.Height

	m.header.SetY(y)
	buf.Merge(m.header.Buffer())

	y += m.header.Height

	m.highlightSelectedRow()
	for _, r := range m.visibleRows() {
		r.SetY(y)
		y += r.GetHeight()
		buf.Merge(r.Buffer())
	}

	return buf
}

//Filter filters the container list by the given filter
func (m *Monitor) Filter(filter string) {

}

//Mount prepares this widget for rendering
func (m *Monitor) Mount() error {
	daemon := m.daemon
	containers := daemon.Containers(
		[]docker.ContainerFilter{docker.ContainerFilters.Running()}, docker.SortByName)
	var rows []*ContainerStatsRow
	var channels []*docker.StatsChannel
	for _, c := range containers {
		statsChan := daemon.OpenChannel(c)
		rows = append(rows, NewSelfUpdatedContainerStatsRow(statsChan, defaultMonitorTableHeader))
		channels = append(channels, statsChan)
	}

	m.rows = rows
	m.openChannels = channels

	m.align()
	return nil
}

//Name returns the name of this widget
func (m *Monitor) Name() string {
	return "Monitor"
}

//OnEvent refreshes the monitor widget. The command is ignored for now.
func (m *Monitor) OnEvent(event EventCommand) error {
	m.refresh()
	return nil
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
				m.refresh()
			}
		}
	}()

}

//RowCount returns the number of rows of this Monitor.
func (m *Monitor) RowCount() int {
	return len(m.rows)
}

//Sort sorts the container list
func (m *Monitor) Sort() {

}

//Unmount tells this widget that it will not be rendering anymore
func (m *Monitor) Unmount() error {
	return nil
}

//Align aligns rows
func (m *Monitor) align() {
	x := m.x
	width := m.width

	m.header.SetWidth(width)
	m.header.SetX(x)

	for _, r := range m.rows {
		r.SetX(x)
		r.SetWidth(width)
	}
}

func (m *Monitor) highlightSelectedRow() {
	if m.RowCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > m.RowCount() {
		index = m.RowCount() - 1
	}

	m.selectedIndex = index
	for i, im := range m.rows {
		if i != index {
			im.NotHighlighted()
		} else {
			im.Highlighted()
		}
	}
}

func (m *Monitor) visibleRows() []*ContainerStatsRow {

	//no screen
	if m.height < 0 {
		return nil
	}
	rows := m.rows
	count := len(rows)
	cursor := ui.ActiveScreen.Cursor
	selected := cursor.Position()
	//everything fits
	if count <= m.height {
		return rows
	}
	//at the the start
	if selected == 0 {
		//internal state is reset
		m.startIndex = 0
		m.endIndex = m.height
		return rows[m.startIndex : m.endIndex+1]
	}

	if selected >= m.endIndex {
		if selected-m.height >= 0 {
			m.startIndex = selected - m.height
		}
		m.endIndex = selected
	}
	if selected <= m.startIndex {
		m.startIndex = m.startIndex - 1
		if selected+m.height < count {
			m.endIndex = m.startIndex + m.height
		}
	}
	start := m.startIndex
	end := m.endIndex + 1
	return rows[start:end]
}
func (m *Monitor) refresh() {
	ui.ActiveScreen.RenderBufferer(m)
	ui.ActiveScreen.Flush()
}
