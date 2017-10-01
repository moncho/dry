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

//ServiceTasksWidget shows a service's task information
type ServiceTasksWidget struct {
	header               *termui.TableHeader
	height, width        int
	info                 *ServiceInfoWidget
	mounted              bool
	offset               int
	serviceID            string
	selectedIndex        int
	sortMode             docker.SortMode
	startIndex, endIndex int
	swarmClient          docker.SwarmAPI
	tableTitle           termui.SizableBufferer
	tasks                []*TaskRow
	x, y                 int
	sync.RWMutex
}

//NewServiceTasksWidget creates a TasksWidget
func NewServiceTasksWidget(swarmClient docker.SwarmAPI, y int) *ServiceTasksWidget {
	w := &ServiceTasksWidget{
		swarmClient:   swarmClient,
		header:        defaultTasksTableHeader,
		mounted:       false,
		offset:        0,
		selectedIndex: 0,
		x:             0,
		y:             y,
		sortMode:      docker.SortByTaskService,
		width:         ui.ActiveScreen.Dimensions.Width}
	return w
}

//Buffer returns the content of this widget as a termui.Buffer
func (s *ServiceTasksWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	y := s.y
	buf := gizaktermui.NewBuffer()
	if s.mounted {
		s.sortRows()

		s.updateHeader()
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
	}
	return buf
}

//ForService sets the service for which this widget is showing tasks
func (s *ServiceTasksWidget) ForService(serviceID string) {
	s.Lock()
	defer s.Unlock()

	s.serviceID = serviceID
	s.mounted = false
}

//Mount prepares this widget for rendering
func (s *ServiceTasksWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if !s.mounted {
		service, err := s.swarmClient.Service(s.serviceID)
		if err != nil {
			return err
		}
		serviceInfo := NewServiceInfoWidget(s.swarmClient, service, s.y)
		s.height = appui.MainScreenAvailableHeight() - serviceInfo.GetHeight()
		s.info = serviceInfo

		tasks, err := s.swarmClient.ServiceTasks(s.serviceID)
		if err != nil {
			return err
		}
		s.tableTitle = createTableTitle(serviceInfo.serviceName, len(tasks))

		rows := make([]*TaskRow, len(tasks))
		for i, task := range tasks {
			rows[i] = NewTaskRow(s.swarmClient, task, s.header)
		}
		s.tasks = rows
		s.align()
		s.mounted = true
	}
	return nil
}

//Name returns this widget name
func (s *ServiceTasksWidget) Name() string {
	return "ServiceTasksWidget"
}

//OnEvent runs the given command
func (s *ServiceTasksWidget) OnEvent(event appui.EventCommand) error {
	return nil
}

//RowCount returns the number of rowns of this widget.
func (s *ServiceTasksWidget) RowCount() int {
	return len(s.tasks)
}

//Sort rotates to the next sort mode.
//SortByTaskImage -> SortByTaskService -> SortByTaskState -> SortByTaskImage
func (s *ServiceTasksWidget) Sort() {
	s.Lock()
	defer s.Unlock()
	switch s.sortMode {
	case docker.SortByTaskImage:
		s.sortMode = docker.SortByTaskService
	case docker.SortByTaskService:
		s.sortMode = docker.SortByTaskState
	case docker.SortByTaskState:
		s.sortMode = docker.SortByTaskImage
	}
}

//Unmount marks this widget as unmounted
func (s *ServiceTasksWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()

	s.mounted = false
	return nil

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

func (s *ServiceTasksWidget) highlightSelectedRow() {
	count := s.RowCount()
	if count == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > count {
		index = count - 1
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

func (s *ServiceTasksWidget) updateHeader() {
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

func (s *ServiceTasksWidget) sortRows() {
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
