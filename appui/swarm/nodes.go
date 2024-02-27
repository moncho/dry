package swarm

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"

	units "github.com/docker/go-units"
	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

// the default length of a widget header
const widgetHeaderLength = 4

var defaultNodeTableHeader = nodeTableHeader()

var nodeTableFieldWidths = map[string]int{
	"NAME":           0,
	"ROLE":           12,
	"LABELS":         0,
	"CPU":            4,
	"MEMORY":         12,
	"DOCKER ENGINE":  16,
	"IP ADDRESS":     16,
	"STATUS":         16,
	"AVAILABILITY":   16,
	"MANAGER STATUS": 0,
}

var nodeTableHeaders = []appui.SortableColumnHeader{
	{Title: "NAME", Mode: appui.SortMode(docker.SortByNodeName)},
	{Title: "ROLE", Mode: appui.SortMode(docker.SortByNodeRole)},
	{Title: "LABELS", Mode: appui.SortMode(docker.NoSortNode)},
	{Title: "CPU", Mode: appui.SortMode(docker.SortByNodeCPU)},
	{Title: "MEMORY", Mode: appui.SortMode(docker.SortByNodeMem)},
	{Title: "DOCKER ENGINE", Mode: appui.SortMode(docker.NoSortNode)},
	{Title: "IP ADDRESS", Mode: appui.SortMode(docker.NoSortNode)},
	{Title: "STATUS", Mode: appui.SortMode(docker.SortByNodeStatus)},
	{Title: "AVAILABILITY", Mode: appui.SortMode(docker.NoSortNode)},
	{Title: "MANAGER STATUS", Mode: appui.SortMode(docker.NoSortNode)},
}

// NodesWidget presents Docker swarm information
type NodesWidget struct {
	swarmClient          docker.SwarmAPI
	filteredRows         []*NodeRow
	totalRows            []*NodeRow
	filterPattern        string
	header               *termui.TableHeader
	selectedIndex        int
	startIndex, endIndex int
	screen               appui.Screen
	sortMode             docker.SortMode
	title                *termui.MarkupPar
	totalMemory          int64
	totalCPU             int

	sync.RWMutex
	mounted bool
}

// NewNodesWidget creates a NodesWidget
func NewNodesWidget(swarmClient docker.SwarmAPI, s appui.Screen) *NodesWidget {
	return &NodesWidget{
		swarmClient: swarmClient,
		header:      defaultNodeTableHeader,
		screen:      s,
		sortMode:    docker.SortByNodeName}
}

// Buffer returns the content of this widget as a termui.Buffer
func (s *NodesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()

	buf := gizaktermui.NewBuffer()
	if !s.mounted {
		return buf
	}
	y := s.screen.Bounds().Min.Y

	s.prepareForRendering()

	widgetHeader := appui.NewWidgetHeader()
	widgetHeader.HeaderEntry("Nodes", strconv.Itoa(s.RowCount()))
	if s.filterPattern != "" {
		widgetHeader.HeaderEntry("Active filter", s.filterPattern)
	}

	widgetHeader.Y = y
	buf.Merge(widgetHeader.Buffer())
	y += widgetHeader.GetHeight()
	//Empty line between the header and the rest of the content
	y++
	s.updateHeader()
	s.header.SetY(y)
	buf.Merge(s.header.Buffer())
	y += s.header.GetHeight()

	selected := s.selectedIndex - s.startIndex

	for i, nodeRow := range s.visibleRows() {
		nodeRow.SetY(y)
		y += nodeRow.GetHeight()
		if i != selected {
			nodeRow.NotHighlighted()
		} else {
			nodeRow.Highlighted()
		}
		buf.Merge(nodeRow.Buffer())
	}

	return buf
}

// Filter applies the given filter to the container list
func (s *NodesWidget) Filter(filter string) {
	s.Lock()
	defer s.Unlock()
	s.filterPattern = filter
}

// Mount prepares this widget for rendering
func (s *NodesWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if s.mounted {
		return nil

	}
	swarmClient := s.swarmClient
	if nodes, err := swarmClient.Nodes(); err == nil {
		docker.SortNodes(nodes, s.sortMode)
		var rows []*NodeRow
		s.totalCPU = 0
		s.totalMemory = 0
		for _, node := range nodes {
			row := NewNodeRow(node, s.header)
			rows = append(rows, row)
			if cpu, err := strconv.Atoi(row.CPU.Text); err == nil {
				s.totalCPU += cpu
			}
			s.totalMemory += node.Description.Resources.MemoryBytes
		}
		s.totalRows = rows
	}
	addSwarmSpecs(s)
	s.align()
	s.mounted = true

	return nil
}

// Name returns this widget name
func (s *NodesWidget) Name() string {
	return "NodesWidget"
}

// OnEvent runs the given command
func (s *NodesWidget) OnEvent(event appui.EventCommand) error {
	if s.RowCount() > 0 {
		return event(s.filteredRows[s.selectedIndex].node.ID)
	}
	return errors.New("the node list is empty")
}

// Unmount tells this widge that in will not be rendered anymore
func (s *NodesWidget) Unmount() error {

	s.Lock()
	defer s.Unlock()
	s.mounted = false
	return nil

}

// RowCount returns the number of rows of this widget.
func (s *NodesWidget) RowCount() int {
	return len(s.filteredRows)
}

// Sort rotates to the next sort mode.
func (s *NodesWidget) Sort() {
	s.RLock()
	defer s.RUnlock()
	switch s.sortMode {
	case docker.SortByNodeName:
		s.sortMode = docker.SortByNodeRole
	case docker.SortByNodeRole:
		s.sortMode = docker.SortByNodeCPU
	case docker.SortByNodeCPU:
		s.sortMode = docker.SortByNodeMem
	case docker.SortByNodeMem:
		s.sortMode = docker.SortByNodeStatus
	case docker.SortByNodeStatus:
		s.sortMode = docker.SortByNodeName
	}
	s.mounted = false
}

// Align aligns rows
func (s *NodesWidget) align() {
	x := s.screen.Bounds().Min.X
	width := s.screen.Bounds().Dx()

	s.title.SetWidth(width)
	s.title.SetX(x)

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, n := range s.totalRows {
		n.SetX(x)
		n.SetWidth(width)
	}
}

func (s *NodesWidget) filterRows() {

	if s.filterPattern != "" {
		var rows []*NodeRow

		for _, row := range s.totalRows {
			if appui.RowFilters.ByPattern(s.filterPattern)(row) {
				rows = append(rows, row)
			}
		}
		s.filteredRows = rows
	} else {
		s.filteredRows = s.totalRows
	}
}

func (s *NodesWidget) calculateVisibleRows() {

	height := s.screen.Bounds().Dy() - widgetHeaderLength
	count := s.RowCount()

	//no screen
	if height < 0 || count == 0 {
		s.startIndex = 0
		s.endIndex = 0
		return
	}
	selected := s.selectedIndex
	//everything fits
	if count <= height {
		s.startIndex = 0
		s.endIndex = count
		return
	}
	//at the the start
	if selected == 0 {
		s.startIndex = 0
		s.endIndex = height
	} else if selected >= count-1 { //at the end
		s.startIndex = count - height
		s.endIndex = count
	} else if selected == s.endIndex { //scroll down by one
		s.startIndex++
		s.endIndex++
	} else if selected <= s.startIndex { //scroll up by one
		s.startIndex--
		s.endIndex--
	} else if selected > s.endIndex { // scroll
		s.startIndex = selected - height
		s.endIndex = selected
	}
}

// prepareForRendering sets the internal state of this widget so it is ready for
// rendering (i.e. Buffer()).
func (s *NodesWidget) prepareForRendering() {
	s.sortRows()
	s.filterRows()
	s.screen.Cursor().Max(s.RowCount() - 1)
	index := s.screen.Cursor().Position()
	if index < 0 {
		index = 0
	} else if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	s.selectedIndex = index
	s.calculateVisibleRows()
}

func (s *NodesWidget) sortRows() {
	rows := s.totalRows
	mode := s.sortMode
	if s.sortMode == docker.NoSortNode {
		return
	}
	var sortAlg func(i, j int) bool
	switch mode {

	case docker.SortByNodeName:
		sortAlg = func(i, j int) bool {
			return rows[i].Name.Text < rows[j].Name.Text
		}
	case docker.SortByNodeRole:
		sortAlg = func(i, j int) bool {
			return rows[i].Role.Text < rows[j].Role.Text
		}
	case docker.SortByNodeCPU:
		sortAlg = func(i, j int) bool {
			return rows[i].CPU.Text < rows[j].CPU.Text
		}
	case docker.SortByNodeMem:
		sortAlg = func(i, j int) bool {
			return rows[i].Memory.Text < rows[j].Memory.Text
		}
	case docker.SortByNodeStatus:
		sortAlg = func(i, j int) bool {
			return rows[i].Status.Text < rows[j].Status.Text
		}
	}
	sort.SliceStable(rows, sortAlg)
}

func (s *NodesWidget) updateHeader() {
	sortMode := s.sortMode

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header appui.SortableColumnHeader
		if strings.Contains(colTitle, appui.DownArrow) {
			colTitle = colTitle[appui.DownArrowLength:]
		}
		for _, h := range nodeTableHeaders {
			if colTitle == h.Title {
				header = h
				break
			}
		}
		if header.Mode == appui.SortMode(sortMode) {
			c.Text = appui.DownArrow + colTitle
		} else {
			c.Text = colTitle
		}

	}

}

func (s *NodesWidget) visibleRows() []*NodeRow {
	return s.filteredRows[s.startIndex:s.endIndex]
}

func nodeTableHeader() *termui.TableHeader {

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	for _, f := range nodeTableHeaders {
		width := nodeTableFieldWidths[f.Title]
		if width == 0 {
			header.AddColumn(f.Title)
		} else {
			header.AddFixedWidthColumn(f.Title, width)
		}
	}
	return header
}

func addSwarmSpecs(w *NodesWidget) {
	par := termui.NewParFromMarkupText(appui.DryTheme,
		strings.Join(
			[]string{
				ui.Blue("Node count:"),
				ui.Yellow(strconv.Itoa(w.RowCount())),
				ui.Blue("Total CPU:"),
				ui.Yellow(strconv.Itoa(w.totalCPU)),
				ui.Blue("Total Memory:"),
				ui.Yellow(units.BytesSize(float64(w.totalMemory))),
			}, " "))
	par.BorderTop = false
	par.BorderBottom = false
	par.BorderLeft = false
	par.BorderRight = false
	par.Height = 1
	par.Bg = gizaktermui.Attribute(appui.DryTheme.Bg)
	par.TextBgColor = gizaktermui.Attribute(appui.DryTheme.Bg)

	w.title = par

}
