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

var nodeTableFields = []string{"NAME",
	"ROLE",
	"LABELS",
	"CPU",
	"MEMORY",
	"DOCKER ENGINE",
	"IP ADDRESS",
	"STATUS",
	"AVAILABILITY",
	"MANAGER STATUS"}

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
		width:         ui.ActiveScreen.Dimensions.Width}
	appui.RegisterWidget(docker.NodeSource, &w)
	return &w
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

//Mount prepares this widget for rendering
func (s *NodesWidget) Mount() error {
	if !s.mounted {
		swarmClient := s.swarmClient
		if nodes, err := swarmClient.Nodes(); err == nil {
			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i].Description.Hostname < nodes[j].Description.Hostname
			})
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

//Unmount tells this widge that in will not be rendered anymore
func (s *NodesWidget) Unmount() error {
	s.mounted = false
	return nil

}

//Buffer returns the content of this widget as a termui.Buffer
func (s *NodesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	buf := gizaktermui.NewBuffer()
	y := s.y

	if s.mounted {
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

//RowCount returns the number of rows of this widget.
func (s *NodesWidget) RowCount() int {
	return len(s.nodes)
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

//OnEvent runs the given command
func (s *NodesWidget) OnEvent(event appui.EventCommand) error {
	if s.RowCount() > 0 {
		return event(s.nodes[s.selectedIndex].node.ID)
	}
	return errors.New("the node list is empty")
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

func nodeTableHeader() *termui.TableHeader {

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	for _, f := range nodeTableFields {
		width := nodeTableFieldWidths[f]
		if width == 0 {
			header.AddColumn(f)
		} else {
			header.AddFixedWidthColumn(f, width)
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
