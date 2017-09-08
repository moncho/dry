package swarm

import (
	"fmt"
	"sync"

	"github.com/docker/docker/api/types/swarm"
	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

var defaultTasksTableHeader = taskTableHeader()

//NodeTasksWidget shows a node's task information
type NodeTasksWidget struct {
	swarmClient          docker.SwarmAPI
	node                 *swarm.Node
	nodeName             string
	tasks                []*TaskRow
	header               *termui.TableHeader
	selectedIndex        int
	offset               int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewNodeTasksWidget creates a TasksWidget
func NewNodeTasksWidget(swarmClient docker.SwarmAPI, y int) *NodeTasksWidget {

	w := NodeTasksWidget{
		swarmClient:   swarmClient,
		header:        defaultTasksTableHeader,
		selectedIndex: 0,
		offset:        0,
		x:             0,
		y:             y,
		height:        appui.MainScreenAvailableHeight(),
		width:         ui.ActiveScreen.Dimensions.Width}

	return &w

}

//PrepareToRender prepares this widget for rendering
func (s *NodeTasksWidget) PrepareToRender(nodeID string) {
	s.Lock()
	defer s.Unlock()
	if node, err := s.swarmClient.Node(nodeID); err == nil {
		nodeName, _ := s.swarmClient.ResolveNode(node.ID)
		s.node = node
		s.nodeName = nodeName

		if tasks, err := s.swarmClient.NodeTasks(node.ID); err == nil {
			for _, task := range tasks {
				s.tasks = append(s.tasks, NewTaskRow(s.swarmClient, task, s.header))
			}
		}
		s.align()

	}
}

//Align aligns rows
func (s *NodeTasksWidget) align() {
	x := s.x
	width := s.width

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, n := range s.tasks {
		n.SetX(x)
		n.SetWidth(width)
	}
}

//Buffer returns the content of this widget as a termui.Buffer
func (s *NodeTasksWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	y := s.y

	buf := gizaktermui.NewBuffer()
	widgetHeader := appui.WidgetHeader(fmt.Sprintf("Node %s task count", s.nodeName), s.RowCount(), "")
	widgetHeader.SetY(y)
	y += widgetHeader.GetHeight()
	buf.Merge(widgetHeader.Buffer())

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

	return buf
}

//RowCount returns the number of rowns of this widget.
func (s *NodeTasksWidget) RowCount() int {
	return len(s.tasks)
}
func (s *NodeTasksWidget) highlightSelectedRow() {
	if s.RowCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	s.tasks[s.selectedIndex].NotHighlighted()
	s.selectedIndex = index
	s.tasks[s.selectedIndex].Highlighted()
}

//OnEvent runs the given command
func (s *NodeTasksWidget) OnEvent(event appui.EventCommand) error {
	return nil
}

func (s *NodeTasksWidget) visibleRows() []*TaskRow {

	//no screen
	if s.height < 0 {
		return nil
	}
	rows := s.tasks
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

func taskTableHeader() *termui.TableHeader {
	fields := []string{
		"TASK ID", "NAME", "IMAGE", "NODE", "DESIRED STATE", "CURRENT STATE", "ERROR", "PORTS"}

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	header.AddFixedWidthColumn(fields[0], docker.ShortLen)
	header.AddColumn(fields[1])
	header.AddColumn(fields[2])
	header.AddColumn(fields[3])
	header.AddFixedWidthColumn(fields[4], 13)
	header.AddFixedWidthColumn(fields[5], 22)
	header.AddColumn(fields[6])
	header.AddColumn(fields[7])

	return header
}
