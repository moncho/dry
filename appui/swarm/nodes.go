package swarm

import (
	"sync"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

var defaultNodeTableHeader = nodeTableHeader()

//NodesWidget presents Docker swarm information
type NodesWidget struct {
	swarmClient          docker.SwarmAPI
	nodes                []*NodeRow
	header               *termui.TableHeader
	selectedIndex        int
	offset               int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewNodesWidget creates a NodesWidget
func NewNodesWidget(swarmClient docker.SwarmAPI, y int) *NodesWidget {
	w := &NodesWidget{
		swarmClient:   swarmClient,
		header:        defaultNodeTableHeader,
		selectedIndex: 0,
		offset:        0,
		x:             0,
		y:             y,
		height:        appui.MainScreenAvailableHeight(),
		width:         ui.ActiveScreen.Dimensions.Width}

	if nodes, err := swarmClient.Nodes(); err == nil {
		for _, node := range nodes {
			w.nodes = append(w.nodes, NewNodeRow(node, w.header))
		}
	}
	w.align()
	return w
}

//Align aligns rows
func (s *NodesWidget) align() {
	y := s.y
	x := s.x
	width := s.width

	s.header.SetWidth(width)
	s.header.SetY(y)
	s.header.SetX(x)

	for _, n := range s.nodes {
		n.SetX(x)
		n.SetWidth(width)
	}
}

//Buffer returns the content of this widget as a termui.Buffer
func (s *NodesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()

	buf := gizaktermui.NewBuffer()
	buf.Merge(s.header.Buffer())

	y := s.y
	y += s.header.GetHeight()

	s.highlightSelectedRow()
	for _, node := range s.visibleRows() {
		node.SetY(y)
		node.Height = 1
		y += node.GetHeight()
		buf.Merge(node.Buffer())
	}

	return buf
}

//RowCount returns the number of rowns of this widget.
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
	return event(s.nodes[s.selectedIndex].node.ID)
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
	fields := []string{
		"NAME", "ROLE", "CPU", "MEMORY", "DOCKER ENGINE", "IP ADDRESS", "STATUS"}

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	for _, f := range fields {
		header.AddColumn(f)
	}
	return header
}
