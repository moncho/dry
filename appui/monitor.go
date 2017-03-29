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
	height := ui.ActiveScreen.Dimensions.Height - MainScreenHeaderSize - MainScreenFooterSize - 5
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
	if m.ContainerCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > m.ContainerCount() {
		index = m.ContainerCount() - 1
	}
	m.rows[m.selectedIndex].NotHighlighted()
	m.selectedIndex = index
	m.rows[m.selectedIndex].Highlighted()
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

//Updates the cursor position in case it is out of bounds
func retrieveCursorPosition(cursor *ui.Cursor, noOfElements int) int {

	if cursor.Position() >= noOfElements {
		cursor.ScrollTo(noOfElements - 1)
	} else if cursor.Position() < 0 {
		cursor.Reset()
	}
	return cursor.Position()
}
