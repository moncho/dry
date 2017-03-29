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
	header        *MonitorTableHeader
	rows          []*ContainerStatsRow
	openChannels  []*docker.StatsChannel
	selectedIndex int
	offset        int
	x, y          int
	height, width int
	sync.RWMutex
}

//NewMonitor creates a new Monitor component that will render itself on the given screen
//at the given position and with the given width.
func NewMonitor(daemon docker.ContainerDaemon, y int) *Monitor {
	height := ui.ActiveScreen.Dimensions.Height - MainScreenHeaderSize - MainScreenFooterSize - 4
	containers := daemon.Containers(docker.ContainerFilters.Running(), docker.SortByName)
	var rows []*ContainerStatsRow
	var channels []*docker.StatsChannel
	for _, c := range containers {
		statsChan := daemon.OpenChannel(c)
		rows = append(rows, NewSelfUpdatedContainerStatsRow(statsChan))
		channels = append(channels, statsChan)
	}
	m := &Monitor{
		header:        defaultMonitorTableHeader,
		rows:          rows,
		openChannels:  channels,
		selectedIndex: 0,
		offset:        0,
		x:             0,
		y:             y,
		height:        height,
		width:         ui.ActiveScreen.Dimensions.Width}
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

	for _, r := range m.rows {
		r.SetX(x)
		r.SetWidth(width)
	}
}

//Buffer returns the content of this monitor as a termui.Buffer
func (m *Monitor) Buffer() gizaktermui.Buffer {
	m.Lock()
	defer m.Unlock()

	buf := gizaktermui.NewBuffer()
	buf.Merge(defaultMonitorTableHeader.Buffer())
	y := m.y
	y += m.header.GetHeight()

	m.highlightSelectedRow()
	for _, r := range m.visibleRows() {
		r.SetY(y)
		y += r.GetHeight()
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
func (m *Monitor) highlightSelectedRow() {
	index := ui.ActiveScreen.Cursor.Position()
	if index > m.ContainerCount() {
		index = m.ContainerCount() - 1
	}
	m.rows[m.selectedIndex].NotHighlighted()
	m.selectedIndex = index
	m.rows[m.selectedIndex].Highlighted()
}

func (m *Monitor) visibleRows() []*ContainerStatsRow {

	availableLines := m.height
	if availableLines < 0 {
		return nil
	}
	rows := m.rows
	rowCount := m.ContainerCount()
	if rowCount < availableLines {
		return rows
	}
	// downwards
	if m.selectedIndex >= m.offset+availableLines {
		m.offset++
	} else if m.selectedIndex < m.offset {
		m.offset--
	}

	if m.offset < 0 {
		m.offset = 0
	} else if m.offset > rowCount-1 {
		m.offset = rowCount - 1
	}

	return m.rows[m.offset : m.offset+availableLines]
}

//Updates the cursor position in case it is out of bounds
func retrieveCursorPosition(cursor *ui.Cursor, noOfElements int) int {

	if cursor.Position() >= noOfElements {
		cursor.ScrollTo(noOfElements - 1)
	} else if cursor.Position() < 0 {
		cursor.Reset()
	}
	return cursor.Position()
}
