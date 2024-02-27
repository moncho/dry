package swarm

import (
	"fmt"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

var defaultTasksTableHeader = taskTableHeader()

// NodeTasksWidget shows a node's task information
type NodeTasksWidget struct {
	TasksWidget
	nodeID   string
	nodeName string
}

// NewNodeTasksWidget creates a TasksWidget
func NewNodeTasksWidget(swarmClient docker.SwarmAPI, s appui.Screen) *NodeTasksWidget {

	w := NodeTasksWidget{
		TasksWidget: TasksWidget{
			swarmClient:   swarmClient,
			header:        defaultTasksTableHeader,
			mounted:       false,
			sortMode:      docker.SortByTaskService,
			selectedIndex: 0,
			offset:        0,
			screen:        s,
			tableTitle:    createStackTableTitle()},
	}

	return &w

}

// Buffer returns the content of this widget as a termui.Buffer
func (s *NodeTasksWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	buf := gizaktermui.NewBuffer()
	if !s.mounted {
		return buf
	}
	y := s.screen.Bounds().Min.Y
	s.prepareForRendering()
	var filter string
	if s.filterPattern != "" {
		filter = fmt.Sprintf(
			"<b><blue> | Active filter: </><yellow>%s</></> ", s.filterPattern)
	}
	s.tableTitle.Content(fmt.Sprintf(
		"<b><blue>Node %s tasks: </><yellow>%d</></>", s.nodeName, s.RowCount()) + " " + filter)

	s.tableTitle.Y = y
	buf.Merge(s.tableTitle.Buffer())
	y += s.tableTitle.GetHeight()

	s.updateHeader()
	s.header.SetY(y)
	buf.Merge(s.header.Buffer())
	y += s.header.GetHeight()

	selected := s.selectedIndex - s.startIndex

	for i, serviceRow := range s.visibleRows() {
		serviceRow.SetY(y)
		y += serviceRow.GetHeight()
		if i != selected {
			serviceRow.NotHighlighted()
		} else {
			serviceRow.Highlighted()
		}
		buf.Merge(serviceRow.Buffer())
	}
	return buf
}

// ForNode sets the node for which this widget will render tasks
func (s *NodeTasksWidget) ForNode(nodeID string) {
	s.Lock()
	defer s.Unlock()
	s.nodeID = nodeID
	s.mounted = false
	s.sortMode = docker.SortByTaskService
}

// Mount prepares this widget for rendering
func (s *NodeTasksWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if s.mounted {
		return nil

	}
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
	s.totalRows = tasksRows
	s.align()
	s.mounted = true

	return nil
}

// Name returns this widget name
func (s *NodeTasksWidget) Name() string {
	return "NodeTasksWidget"
}
