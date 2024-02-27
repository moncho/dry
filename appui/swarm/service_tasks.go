package swarm

import (
	"fmt"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

// ServiceTasksWidget shows a service's task information
type ServiceTasksWidget struct {
	serviceID string
	info      *ServiceInfoWidget
	TasksWidget
}

// NewServiceTasksWidget creates a TasksWidget
func NewServiceTasksWidget(swarmClient docker.SwarmAPI, s appui.Screen) *ServiceTasksWidget {
	w := &ServiceTasksWidget{
		TasksWidget: TasksWidget{
			swarmClient:   swarmClient,
			header:        defaultTasksTableHeader,
			mounted:       false,
			offset:        0,
			selectedIndex: 0,
			screen:        s,
			sortMode:      docker.SortByTaskService,
			tableTitle:    createStackTableTitle()},
	}
	return w
}

// Buffer returns the content of this widget as a termui.Buffer
func (s *ServiceTasksWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	buf := gizaktermui.NewBuffer()
	if !s.mounted {
		return buf
	}
	y := s.screen.Bounds().Min.Y
	s.prepareForRendering()
	s.info.SetY(y)
	buf.Merge(s.info.Buffer())
	y += s.info.GetHeight()
	var filter string
	if s.filterPattern != "" {
		filter = fmt.Sprintf(
			"<b><blue> | Active filter: </><yellow>%s</></> ", s.filterPattern)
	}
	s.tableTitle.Content(fmt.Sprintf(
		"<b><blue>Service %s tasks: </><yellow>%d</></>", s.info.serviceName, s.RowCount()) + " " + filter)

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

// ForService sets the service for which this widget is showing tasks
func (s *ServiceTasksWidget) ForService(serviceID string) {
	s.Lock()
	defer s.Unlock()
	s.serviceID = serviceID
	s.mounted = false
	s.sortMode = docker.SortByTaskService
}

// Mount prepares this widget for rendering
func (s *ServiceTasksWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if s.mounted {
		return nil
	}
	service, err := s.swarmClient.Service(s.serviceID)
	if err != nil {
		return err
	}
	serviceInfo := NewServiceInfoWidget(s.swarmClient, service, s.screen)
	s.info = serviceInfo

	tasks, err := s.swarmClient.ServiceTasks(s.serviceID)
	if err != nil {
		return err
	}

	rows := make([]*TaskRow, len(tasks))
	for i, task := range tasks {
		rows[i] = NewTaskRow(s.swarmClient, task, s.header)
	}
	s.totalRows = rows

	s.align()
	s.mounted = true
	return nil
}

// Name returns this widget name
func (s *ServiceTasksWidget) Name() string {
	return "ServiceTasksWidget"
}
