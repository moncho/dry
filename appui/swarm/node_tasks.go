package swarm

import (
	"fmt"
	"sort"
	"strings"
	"sync"

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
	mounted              bool
	nodeID               string
	nodeName             string
	tasks                []*TaskRow
	header               *termui.TableHeader
	selectedIndex        int
	offset               int
	x, y                 int
	height, width        int
	sortMode             docker.SortMode
	startIndex, endIndex int
	sync.RWMutex
}

//NewNodeTasksWidget creates a TasksWidget
func NewNodeTasksWidget(swarmClient docker.SwarmAPI, y int) *NodeTasksWidget {

	w := NodeTasksWidget{
		swarmClient:   swarmClient,
		header:        defaultTasksTableHeader,
		mounted:       false,
		sortMode:      docker.SortByTaskService,
		selectedIndex: 0,
		offset:        0,
		x:             0,
		y:             y,
		height:        appui.MainScreenAvailableHeight(),
		width:         ui.ActiveScreen.Dimensions.Width}

	return &w

}

//Buffer returns the content of this widget as a termui.Buffer
func (s *NodeTasksWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	y := s.y
	buf := gizaktermui.NewBuffer()
	if s.mounted {
		s.sortRows()
		s.updateHeader()
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
	}
	return buf
}

//ForNode sets the node for which this widget will render tasks
func (s *NodeTasksWidget) ForNode(nodeID string) {
	s.Lock()
	defer s.Unlock()
	s.nodeID = nodeID
	s.mounted = false
	s.sortMode = docker.SortByTaskService
}

//Mount prepares this widget for rendering
func (s *NodeTasksWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if !s.mounted {
		node, err := s.swarmClient.Node(s.nodeID)
		if err != nil {
			return err
		}
		nodeName, _ := s.swarmClient.ResolveNode(node.ID)

		s.nodeName = nodeName

		tasks, err := s.swarmClient.NodeTasks(node.ID)
		if err != nil {

			return err
		}

		var tasksRows []*TaskRow
		for _, task := range tasks {
			tasksRows = append(tasksRows, NewTaskRow(s.swarmClient, task, s.header))
		}
		s.tasks = tasksRows
		s.align()
		s.mounted = true

	}
	return nil
}

//Name returns this widget name
func (s *NodeTasksWidget) Name() string {
	return "NodeTasksWidget"

}

//OnEvent runs the given command
func (s *NodeTasksWidget) OnEvent(event appui.EventCommand) error {
	return nil
}

//RowCount returns the number of rowns of this widget.
func (s *NodeTasksWidget) RowCount() int {
	return len(s.tasks)
}

//Sort rotates to the next sort mode.
//SortByTaskService -> SortByTaskImage -> SortByTaskState -> SortByTaskService
func (s *NodeTasksWidget) Sort() {
	s.Lock()
	defer s.Unlock()
	switch s.sortMode {
	case docker.SortByTaskService:
		s.sortMode = docker.SortByTaskImage
	case docker.SortByTaskImage:
		s.sortMode = docker.SortByTaskState
	case docker.SortByTaskState:
		s.sortMode = docker.SortByTaskService
	}
}

//Unmount marks this widget as unmounted
func (s *NodeTasksWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()
	s.mounted = false
	return nil
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

func (s *NodeTasksWidget) highlightSelectedRow() {
	if s.RowCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	s.selectedIndex = index
	for i, row := range s.tasks {
		if i != index {
			row.NotHighlighted()
		} else {
			row.Highlighted()
		}
	}
}

func (s *NodeTasksWidget) sortRows() {
	rows := s.tasks
	mode := s.sortMode
	if s.sortMode == docker.NoSortTask {
		return
	}
	var sortAlg func(i, j int) bool
	switch mode {
	case docker.SortByTaskImage:
		sortAlg = func(i, j int) bool {
			return rows[i].Image.Text < rows[j].Image.Text
		}
	case docker.SortByTaskService:
		sortAlg = func(i, j int) bool {
			return rows[i].Name.Text < rows[j].Name.Text
		}
	case docker.SortByTaskState:
		sortAlg = func(i, j int) bool {
			return rows[i].CurrentState.Text < rows[j].CurrentState.Text
		}
	}
	sort.SliceStable(rows, sortAlg)
}

func (s *NodeTasksWidget) updateHeader() {
	sortMode := s.sortMode

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header appui.SortableColumnHeader
		if strings.Contains(colTitle, appui.DownArrow) {
			colTitle = colTitle[appui.DownArrowLength:]
		}
		for _, h := range taskTableHeaders {
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

var taskTableHeaders = []appui.SortableColumnHeader{
	{Title: "TASK ID", Mode: docker.NoSortTask},
	{Title: "NAME", Mode: docker.SortByTaskService},
	{Title: "IMAGE", Mode: docker.SortByTaskImage},
	{Title: "NODE", Mode: docker.NoSortTask},
	{Title: "DESIRED STATE", Mode: docker.NoSortTask},
	{Title: "CURRENT STATE", Mode: docker.SortByTaskState},
	{Title: "ERROR", Mode: docker.NoSortTask},
	{Title: "PORTS", Mode: docker.NoSortTask},
}

func taskTableHeader() *termui.TableHeader {

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	header.AddFixedWidthColumn(taskTableHeaders[0].Title, docker.ShortLen)
	header.AddColumn(taskTableHeaders[1].Title)
	header.AddColumn(taskTableHeaders[2].Title)
	header.AddColumn(taskTableHeaders[3].Title)
	header.AddFixedWidthColumn(taskTableHeaders[4].Title, 13)
	header.AddFixedWidthColumn(taskTableHeaders[5].Title, 22)
	header.AddColumn(taskTableHeaders[6].Title)
	header.AddColumn(taskTableHeaders[7].Title)

	return header
}
