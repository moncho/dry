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

//ServiceTasksWidget shows a service's task information
type ServiceTasksWidget struct {
	swarmClient          docker.SwarmAPI
	service              *swarm.Service
	tasks                []*TaskRow
	header               *termui.TableHeader
	info                 *ServiceInfoWidget
	tableTitle           termui.SizableBufferer
	selectedIndex        int
	offset               int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewServiceTasksWidget creates a TasksWidget
func NewServiceTasksWidget(swarmClient docker.SwarmAPI, y int) *ServiceTasksWidget {
	w := &ServiceTasksWidget{
		swarmClient:   swarmClient,
		header:        defaultTasksTableHeader,
		selectedIndex: 0,
		offset:        0,
		x:             0,
		y:             y,
		width:         ui.ActiveScreen.Dimensions.Width}
	return w
}

//PrepareToRender prepares this widget for rendering
func (s *ServiceTasksWidget) PrepareToRender(serviceID string) {
	s.Lock()
	defer s.Unlock()
	if service, err := s.swarmClient.Service(serviceID); err == nil {
		serviceInfo := NewServiceInfoWidget(s.swarmClient, service, s.y)
		s.height = appui.MainScreenAvailableHeight() - serviceInfo.GetHeight()
		s.service = service
		s.info = serviceInfo

		if tasks, err := s.swarmClient.ServiceTasks(serviceID); err == nil {
			s.tableTitle = createTableTitle(serviceInfo.serviceName, len(tasks))
			var rows []*TaskRow
			for _, task := range tasks {
				rows = append(rows, NewTaskRow(s.swarmClient, task, s.header))
			}
			s.tasks = rows
		}
		s.align()
	}

}

//Align aligns rows
func (s *ServiceTasksWidget) align() {
	x := s.x
	width := s.width

	s.info.SetX(x)
	s.info.SetWidth(width)
	s.tableTitle.SetX(x)
	s.tableTitle.SetWidth(width)
	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, n := range s.tasks {
		n.SetX(x)
		n.SetWidth(width)
	}
}

//Buffer returns the content of this widget as a termui.Buffer
func (s *ServiceTasksWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	y := s.y

	buf := gizaktermui.NewBuffer()
	if s.info != nil {
		buf.Merge(s.info.Buffer())
		y += s.info.GetHeight()
	}
	s.tableTitle.SetY(y)
	buf.Merge(s.tableTitle.Buffer())
	y += s.tableTitle.GetHeight()
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
func (s *ServiceTasksWidget) RowCount() int {
	return len(s.tasks)
}
func (s *ServiceTasksWidget) highlightSelectedRow() {
	count := s.RowCount()
	if count == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > count {
		index = count - 1
	}
	if s.selectedIndex < count && s.tasks[s.selectedIndex] != nil {
		s.tasks[s.selectedIndex].NotHighlighted()
	}
	s.selectedIndex = index
	s.tasks[s.selectedIndex].Highlighted()
}

//OnEvent runs the given command
func (s *ServiceTasksWidget) OnEvent(event appui.EventCommand) error {
	return nil
}

func (s *ServiceTasksWidget) visibleRows() []*TaskRow {

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

func createTableTitle(serviceName string, count int) termui.SizableBufferer {
	p := termui.NewKeyValuePar(
		fmt.Sprintf("Service %s tasks", serviceName),
		fmt.Sprintf("%d", count),
		appui.DryTheme)
	return p
}
