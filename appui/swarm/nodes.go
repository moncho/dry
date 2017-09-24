package swarm

import (
	"errors"
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
	{Title: "NAME", Mode: docker.SortByNodeName},
	{Title: "ROLE", Mode: docker.SortByNodeRole},
	{Title: "LABELS", Mode: docker.NoSortNode},
	{Title: "CPU", Mode: docker.SortByNodeCPU},
	{Title: "MEMORY", Mode: docker.SortByNodeMem},
	{Title: "DOCKER ENGINE", Mode: docker.NoSortNode},
	{Title: "IP ADDRESS", Mode: docker.NoSortNode},
	{Title: "STATUS", Mode: docker.SortByNodeStatus},
	{Title: "AVAILABILITY", Mode: docker.NoSortNode},
	{Title: "MANAGER STATUS", Mode: docker.NoSortNode},
}

//NodesWidget presents Docker swarm information
type NodesWidget struct {
	swarmClient          docker.SwarmAPI
	nodes                []*NodeRow
	header               *termui.TableHeader
	title                *termui.MarkupPar
	selectedIndex        int
	offset               int
	x, y                 int
	height, width        int
	mounted              bool
	sortMode             docker.SortMode
	startIndex, endIndex int
	totalMemory          int64
	totalCPU             int
	sync.RWMutex
}

//NewNodesWidget creates a NodesWidget
func NewNodesWidget(swarmClient docker.SwarmAPI, y int) *NodesWidget {
	w := NodesWidget{
		swarmClient:   swarmClient,
		header:        defaultNodeTableHeader,
		selectedIndex: 0,
		offset:        0,
		x:             0,
		y:             y,
		height:        appui.MainScreenAvailableHeight(),
		width:         ui.ActiveScreen.Dimensions.Width,
		sortMode:      docker.SortByNodeName}
	appui.RegisterWidget(docker.NodeSource, &w)
	return &w
}

//Buffer returns the content of this widget as a termui.Buffer
func (s *NodesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	buf := gizaktermui.NewBuffer()
	y := s.y

	if s.mounted {
		s.updateHeader()
		s.title.Y = y
		buf.Merge(s.title.Buffer())
		y += s.title.GetHeight() + 1
		s.header.SetY(y)
		buf.Merge(s.header.Buffer())
		y += s.header.GetHeight()

		s.highlightSelectedRow()
		for _, node := range s.visibleRows() {
			node.SetY(y)
			node.Height = 1
			y += node.GetHeight()
			buf.Merge(node.Buffer())
		}
	}

	return buf
}

//Mount prepares this widget for rendering
func (s *NodesWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if !s.mounted {
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
			s.nodes = rows
		}
		addSwarmSpecs(s)
		s.align()
		s.mounted = true
	}
	return nil
}

//Name returns this widget name
func (s *NodesWidget) Name() string {
	return "NodesWidget"
}

//OnEvent runs the given command
func (s *NodesWidget) OnEvent(event appui.EventCommand) error {
	if s.RowCount() > 0 {
		return event(s.nodes[s.selectedIndex].node.ID)
	}
	return errors.New("the node list is empty")
}

//Unmount tells this widge that in will not be rendered anymore
func (s *NodesWidget) Unmount() error {

	s.Lock()
	defer s.Unlock()
	s.mounted = false
	return nil

}

//RowCount returns the number of rows of this widget.
func (s *NodesWidget) RowCount() int {
	return len(s.nodes)
}

//Sort rotates to the next sort mode.
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

//Align aligns rows
func (s *NodesWidget) align() {
	x := s.x
	width := s.width

	s.title.SetWidth(width)
	s.title.SetX(x)

	s.header.SetWidth(width)

	s.header.SetX(x)

	for _, n := range s.nodes {
		n.SetX(x)
		n.SetWidth(width)
	}
}

func (s *NodesWidget) highlightSelectedRow() {
	if s.RowCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	s.nodes[s.selectedIndex].NotHighlighted()
	s.selectedIndex = index
	s.nodes[s.selectedIndex].Highlighted()
}

func (s *NodesWidget) visibleRows() []*NodeRow {

	//no screen
	if s.height < 0 {
		return nil
	}
	rows := s.nodes
	count := len(rows)
	cursor := ui.ActiveScreen.Cursor
	selected := cursor.Position()
	//everything fits
	if count <= s.height {
		return rows
	}
	//at the the start
	if selected == 0 {
		//internal state is reset
		s.startIndex = 0
		s.endIndex = s.height
		return rows[s.startIndex : s.endIndex+1]
	}

	if selected >= s.endIndex {
		if selected-s.height >= 0 {
			s.startIndex = selected - s.height
		}
		s.endIndex = selected
	}
	if selected <= s.startIndex {
		s.startIndex = s.startIndex - 1
		if selected+s.height < count {
			s.endIndex = s.startIndex + s.height
		}
	}
	start := s.startIndex
	end := s.endIndex + 1
	return rows[start:end]
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
		if header.Mode == sortMode {
			c.Text = appui.DownArrow + colTitle
		} else {
			c.Text = colTitle
		}

	}

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
