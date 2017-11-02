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

//StacksTasksWidget shows a service's task information
type StacksTasksWidget struct {
	header               *termui.TableHeader
	filteredRows         []*TaskRow
	totalRows            []*TaskRow
	filterPattern        string
	height, width        int
	mounted              bool
	offset               int
	stack                string
	selectedIndex        int
	sortMode             docker.SortMode
	startIndex, endIndex int
	swarmClient          docker.SwarmAPI
	tableTitle           *termui.MarkupPar
	x, y                 int
	sync.RWMutex
}

//NewStacksTasksWidget creates a StacksTasksWidget
func NewStacksTasksWidget(swarmClient docker.SwarmAPI, y int) *StacksTasksWidget {
	w := StacksTasksWidget{
		swarmClient:   swarmClient,
		header:        defaultTasksTableHeader,
		height:        appui.MainScreenAvailableHeight(),
		mounted:       false,
		offset:        0,
		selectedIndex: 0,
		x:             0,
		y:             y,
		sortMode:      docker.SortByTaskService,
		tableTitle:    createStackTableTitle(),
		width:         ui.ActiveScreen.Dimensions.Width}
	return &w
}

//Buffer returns the content of this widget as a termui.Buffer
func (s *StacksTasksWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	y := s.y
	buf := gizaktermui.NewBuffer()
	if s.mounted {
		s.prepareForRendering()
		var filter string
		if s.filterPattern != "" {
			filter = fmt.Sprintf(
				"<b><blue> | Active filter: </><yellow>%s</></> ", s.filterPattern)
		}
		s.tableTitle.Content(fmt.Sprintf(
			"<b><blue>Stack %s tasks: </><yellow>%d</></>", s.stack, s.RowCount()) + " " + filter)

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
	}
	return buf
}

//Filter applies the given filter to the container list
func (s *StacksTasksWidget) Filter(filter string) {
	s.Lock()
	defer s.Unlock()
	s.filterPattern = filter
}

//ForStack sets the stack for which this widget is showing tasks
func (s *StacksTasksWidget) ForStack(stack string) {
	s.Lock()
	defer s.Unlock()

	s.stack = stack
	s.mounted = false
	s.sortMode = docker.SortByTaskService

}

//Mount prepares this widget for rendering
func (s *StacksTasksWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if !s.mounted {
		s.mounted = true
		var rows []*TaskRow
		if tasks, err := s.swarmClient.StackTasks(s.stack); err == nil {
			for _, task := range tasks {
				rows = append(rows, NewTaskRow(s.swarmClient, task, s.header))
			}
			s.totalRows = rows
		} else {
			return err
		}
	}
	s.align()
	return nil
}

//Name returns this widget name
func (s *StacksTasksWidget) Name() string {
	return "StacksTasksWidget"
}

//OnEvent runs the given command
func (s *StacksTasksWidget) OnEvent(event appui.EventCommand) error {
	return nil
}

//RowCount returns the number of rowns of this widget.
func (s *StacksTasksWidget) RowCount() int {
	return len(s.filteredRows)
}

//Sort rotates to the next sort mode.
//SortByTaskService -> SortByTaskImage -> SortByTaskDesiredState -> SortByTaskState -> SortByTaskService
func (s *StacksTasksWidget) Sort() {
	s.Lock()
	defer s.Unlock()
	switch s.sortMode {
	case docker.SortByTaskService:
		s.sortMode = docker.SortByTaskImage
	case docker.SortByTaskImage:
		s.sortMode = docker.SortByTaskDesiredState
	case docker.SortByTaskDesiredState:
		s.sortMode = docker.SortByTaskState
	case docker.SortByTaskState:
		s.sortMode = docker.SortByTaskService
	}
}

//Unmount marks this widget as unmounted
func (s *StacksTasksWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()

	s.mounted = false
	return nil

}

//Align aligns rows
func (s *StacksTasksWidget) align() {
	x := s.x
	width := s.width

	s.tableTitle.SetX(x)
	s.tableTitle.SetWidth(width)

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, task := range s.totalRows {
		task.SetX(x)
		task.SetWidth(width)
	}

}

func (s *StacksTasksWidget) filterRows() {

	if s.filterPattern != "" {
		var rows []*TaskRow

		for _, row := range s.totalRows {
			if appui.RowFilters.ByPattern(s.filterPattern)(row) {
				rows = append(rows, row)
			}
		}
		s.filteredRows = rows
	} else {
		s.filteredRows = s.totalRows
	}
}

func (s *StacksTasksWidget) calculateVisibleRows() {

	count := s.RowCount()

	//no screen
	if s.height < 0 || count == 0 {
		s.startIndex = 0
		s.endIndex = 0
		return
	}
	selected := s.selectedIndex
	//everything fits
	if count <= s.height {
		s.startIndex = 0
		s.endIndex = count
		return
	}
	//at the the start
	if selected == 0 {
		s.startIndex = 0
		s.endIndex = s.height
	} else if selected >= count-1 { //at the end
		s.startIndex = count - s.height
		s.endIndex = count
	} else if selected == s.endIndex { //scroll down by one
		s.startIndex++
		s.endIndex++
	} else if selected <= s.startIndex { //scroll up by one
		s.startIndex--
		s.endIndex--
	} else if selected > s.endIndex { // scroll
		s.startIndex = selected - s.height
		s.endIndex = selected
	}
}

//prepareForRendering sets the internal state of this widget so it is ready for
//rendering (i.e. Buffer()).
func (s *StacksTasksWidget) prepareForRendering() {
	s.sortRows()
	s.filterRows()
	index := ui.ActiveScreen.Cursor.Position()
	if index < 0 {
		index = 0
	} else if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	s.selectedIndex = index
	s.calculateVisibleRows()
}

func (s *StacksTasksWidget) updateHeader() {
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

func (s *StacksTasksWidget) visibleRows() []*TaskRow {
	return s.filteredRows[s.startIndex:s.endIndex]
}

func createStackTableTitle() *termui.MarkupPar {
	p := termui.NewParFromMarkupText(appui.DryTheme, "")
	p.Bg = gizaktermui.Attribute(appui.DryTheme.Bg)
	p.TextBgColor = gizaktermui.Attribute(appui.DryTheme.Bg)
	p.TextFgColor = gizaktermui.Attribute(appui.DryTheme.Info)
	p.Border = false

	return p
}

func (s *StacksTasksWidget) sortRows() {
	rows := s.totalRows
	mode := s.sortMode
	if mode == docker.NoSortTask {
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
	case docker.SortByTaskDesiredState:
		sortAlg = func(i, j int) bool {
			return rows[i].DesiredState.Text < rows[j].DesiredState.Text
		}

	}
	sort.SliceStable(rows, sortAlg)
}
