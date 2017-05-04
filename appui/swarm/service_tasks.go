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

//ServiceTasksWidget shows a service's task information
type ServiceTasksWidget struct {
	swarmClient          docker.SwarmAPI
	service              *swarm.Service
	tasks                []*TaskRow
	header               *termui.TableHeader
	selectedIndex        int
	offset               int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewServiceTasksWidget creates a TasksWidget
func NewServiceTasksWidget(swarmClient docker.SwarmAPI, serviceID string, y int) *ServiceTasksWidget {
	if service, err := swarmClient.Service(serviceID); err == nil {

		w := &ServiceTasksWidget{
			swarmClient:   swarmClient,
			service:       service,
			header:        defaultTasksTableHeader,
			selectedIndex: 0,
			offset:        0,
			x:             0,
			y:             y,
			height:        appui.MainScreenAvailableHeight(),
			width:         ui.ActiveScreen.Dimensions.Width}

		if tasks, err := swarmClient.ServiceTasks(serviceID); err == nil {
			for _, task := range tasks {
				w.tasks = append(w.tasks, NewTaskRow(swarmClient, task, w.header))
			}
		}
		w.align()
		return w
	}
	return nil
}

//Align aligns rows
func (s *ServiceTasksWidget) align() {
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
func (s *ServiceTasksWidget) Buffer() gizaktermui.Buffer {
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
func (s *ServiceTasksWidget) RowCount() int {
	return len(s.tasks)
}
func (s *ServiceTasksWidget) highlightSelectedRow() {
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
