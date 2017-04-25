package swarm

import (
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
	tasks                []*TaskRow
	header               *termui.TableHeader
	selectedIndex        int
	offset               int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewTasksWidget creates a TasksWidget
func NewTasksWidget(swarmClient docker.SwarmAPI, nodeID string, y int) *NodeTasksWidget {
	if node, err := swarmClient.Node(nodeID); err == nil {

		w := &NodeTasksWidget{
			swarmClient:   swarmClient,
			node:          node,
			header:        defaultTasksTableHeader,
			selectedIndex: 0,
			offset:        0,
			x:             0,
			y:             y,
			height:        appui.MainScreenAvailableHeight(),
			width:         ui.ActiveScreen.Dimensions.Width}

		if tasks, err := swarmClient.Tasks(node.ID); err == nil {
			for _, task := range tasks {
				w.tasks = append(w.tasks, NewTaskRow(task))
			}
		}
		w.align()
		return w
	}
	return nil
}

//Align aligns rows
func (s *NodeTasksWidget) align() {
	y := s.y
	x := s.x
	width := s.width

	s.header.SetWidth(width)
	s.header.SetY(y)
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
		"ID", "NAME", "IMAGE", "NODE", "DESIRED STATE", "CURRENT STATE", "ERROR", "PORTS"}

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	for _, f := range fields {
		header.AddColumn(f)
	}
	return header
}
