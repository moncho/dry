package appui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

const (
	id sortMode = iota
	name
	cpu
	mem
	netio
	blockio
	pids
	uptime
)

var monitorTableHeaders = map[string]sortMode{
	"CONTAINER": id,
	"NAME":      name,
	"CPU":       cpu,
	"MEM":       mem,
	"NET RX/TX": netio,
	"BLOCK I/O": blockio,
	"PIDS":      pids,
	"UPTIME":    uptime,
}

var defaultRefreshRate = 500 * time.Millisecond

//Monitor is a self-refreshing ui component that shows monitoring information about docker
//containers.
type Monitor struct {
	sync.RWMutex

	cancel               context.CancelFunc
	daemon               docker.ContainerDaemon
	header               *MonitorTableHeader
	height, width        int
	offset               int
	openChannels         []*docker.StatsChannel
	refreshRate          time.Duration
	rowChannels          map[*ContainerStatsRow]*docker.StatsChannel
	rows                 []*ContainerStatsRow
	startIndex, endIndex int
	screen               ScreenBuffererRender
	selectedIndex        int
	sortMode             sortMode
	x, y                 int
}

//NewMonitor creates a new Monitor component that will render itself on the given screen
//at the given position and with the given width.
func NewMonitor(daemon docker.ContainerDaemon, s ScreenBuffererRender, y int) *Monitor {
	height := MainScreenAvailableHeight(s)
	m := Monitor{
		header:        defaultMonitorTableHeader,
		daemon:        daemon,
		selectedIndex: 0,
		offset:        0,
		x:             0,
		y:             y,
		height:        height,
		width:         ui.ActiveScreen.Dimensions().Width,
		refreshRate:   defaultRefreshRate,
		screen:        s,
		sortMode:      id,
	}
	return &m
}

//Buffer returns the content of this monitor as a termui.Buffer
func (m *Monitor) Buffer() gizaktermui.Buffer {
	m.RLock()
	defer m.RUnlock()
	y := m.y
	buf := gizaktermui.NewBuffer()
	widgetHeader := NewWidgetHeader()
	widgetHeader.HeaderEntry("Running Containers", strconv.Itoa(m.RowCount()))
	widgetHeader.HeaderEntry("Refresh rate", m.refreshRate.String())

	widgetHeader.Y = y

	buf.Merge(widgetHeader.Buffer())
	y += widgetHeader.GetHeight()
	//Empty line between the header and the rest of the content
	y++

	m.header.SetY(y)
	buf.Merge(m.header.Buffer())

	y += m.header.Height

	m.highlightSelectedRow()
	m.sortRows()
	for _, r := range m.visibleRows() {
		r.SetY(y)
		y += r.GetHeight()
		buf.Merge(r.Buffer())
	}

	return buf
}

//Filter filters the container list by the given filter
func (m *Monitor) Filter(_ string) {

}

//Mount prepares this widget for rendering
func (m *Monitor) Mount() error {

	if m.cancel != nil {
		return nil
	}
	m.Lock()
	defer m.Unlock()
	daemon := m.daemon
	rowChannels := make(map[*ContainerStatsRow]*docker.StatsChannel)
	containers := daemon.Containers(
		[]docker.ContainerFilter{docker.ContainerFilters.Running()}, docker.SortByName)
	var rows []*ContainerStatsRow
	var channels []*docker.StatsChannel
	for _, c := range containers {
		statsChan, err := daemon.StatsChannel(c)
		if err != nil {
			return errors.Wrap(err, "error mounting monitor widget")
		}
		row := NewContainerStatsRow(c, defaultMonitorTableHeader)
		rows = append(rows, row)
		channels = append(channels, statsChan)
		rowChannels[row] = statsChan
	}

	m.rows = rows
	m.openChannels = channels
	m.rowChannels = rowChannels

	m.align()
	m.updateTableHeader()
	ctx, cancel := context.WithCancel(context.Background())
	m.refreshLoop(ctx)
	m.cancel = cancel
	return nil
}

//Name returns the name of this widget
func (m *Monitor) Name() string {
	return "Monitor"
}

//OnEvent refreshes the monitor widget and runs the given
//command on the highlighted row.
//It can be used to refresh the widget.
func (m *Monitor) OnEvent(event EventCommand) error {
	m.refresh()
	if event == nil {
		return nil
	}
	rows := len(m.visibleRows())
	if rows < 0 {
		return errors.New("there are no rows")
	}

	if m.selectedIndex >= rows {
		return fmt.Errorf("there is no row on index %d", m.selectedIndex)
	}

	return event(m.visibleRows()[m.selectedIndex].container.ID)
}

//RefreshRate sets the refresh rate of this monitor to the given amount in
//milliseconds.
func (m *Monitor) RefreshRate(millis int) {
	m.Lock()
	defer m.Unlock()
	m.refreshRate = time.Duration(millis) * time.Millisecond
}

//refreshLoop signals this monitor to refresh itself until the given context is cancelled
func (m *Monitor) refreshLoop(ctx context.Context) {
	go func(rowChannels map[*ContainerStatsRow]*docker.StatsChannel) {

		for row, ch := range rowChannels {
			stats := ch.Start(ctx)
			go func(row *ContainerStatsRow) {
				for stat := range stats {
					row.Update(stat)
				}
				row.markAsNotRunning()
			}(row)
		}
		m.refresh()
		refreshTimer := time.NewTicker(m.refreshRate)
		for {
			select {
			case <-ctx.Done():
				refreshTimer.Stop()
				return
			case <-refreshTimer.C:
				m.refresh()
			}
		}

	}(m.rowChannels)
}

//RowCount returns the number of rows of this Monitor.
func (m *Monitor) RowCount() int {
	return len(m.rows)
}

//Sort sorts the container list
func (m *Monitor) Sort() {
	m.Lock()
	defer m.Unlock()
	if m.sortMode == uptime {
		m.sortMode = id
	} else {
		m.sortMode++
	}

	m.updateTableHeader()
}

//Unmount tells this widget that it will not be rendering anymore
func (m *Monitor) Unmount() error {
	m.Lock()
	m.cancel()
	m.cancel = nil
	defer m.Unlock()
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
	index := m.screen.Cursor().Position()
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
func (m *Monitor) refresh() {
	m.screen.RenderBufferer(m)
	m.screen.Flush()
}

func (m *Monitor) sortRows() {
	rows := m.rows
	mode := m.sortMode

	var sortAlg func(i, j int) bool

	switch mode {
	case id:
		sortAlg = func(i, j int) bool {
			return rows[i].ID.Text > rows[j].ID.Text
		}
	case name:
		sortAlg = func(i, j int) bool {
			return rows[i].Name.Text > rows[j].Name.Text
		}
	case cpu:
		sortAlg = func(i, j int) bool {
			return rows[i].CPU.Percent > rows[j].CPU.Percent
		}
	case mem:
		sortAlg = func(i, j int) bool {
			return rows[i].Memory.Percent > rows[j].Memory.Percent
		}
	case netio:
		sortAlg = func(i, j int) bool {
			return rows[i].Net.Text > rows[j].Net.Text
		}
	case blockio:
		sortAlg = func(i, j int) bool {
			return rows[i].Block.Text > rows[j].Block.Text
		}
	case pids:
		sortAlg = func(i, j int) bool {
			return rows[i].PidsVal > rows[j].PidsVal
		}
	case uptime:
		sortAlg = func(i, j int) bool {
			return rows[i].UptimeVal.After(rows[j].UptimeVal)
		}
	}
	sort.SliceStable(rows, sortAlg)
}

func (m *Monitor) updateTableHeader() {

	for _, c := range m.header.Columns {
		colTitle := c.Text
		if strings.Contains(colTitle, DownArrow) {
			colTitle = colTitle[DownArrowLength:]
			c.Text = colTitle
		}
		if sortMode, ok := monitorTableHeaders[colTitle]; ok && sortMode == m.sortMode {
			c.Text = DownArrow + colTitle
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
	cursor := m.screen.Cursor()
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
