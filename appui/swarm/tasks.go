package swarm

import (
	"sort"
	"strings"
	"sync"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui/termui"
)

// TasksWidget shows a service's task information
type TasksWidget struct {
	header               *termui.TableHeader
	filteredRows         []*TaskRow
	totalRows            []*TaskRow
	filterPattern        string
	offset               int
	screen               appui.Screen
	selectedIndex        int
	sortMode             docker.SortMode
	startIndex, endIndex int
	swarmClient          docker.SwarmAPI
	tableTitle           *termui.MarkupPar

	sync.RWMutex
	mounted bool
}

// Filter applies the given filter to the container list
func (s *TasksWidget) Filter(filter string) {
	s.Lock()
	defer s.Unlock()
	s.filterPattern = filter
}

// OnEvent runs the given command
func (s *TasksWidget) OnEvent(event appui.EventCommand) error {
	if s.RowCount() > 0 {
		return event(s.filteredRows[s.selectedIndex].task.ID)
	}
	return nil
}

// RowCount returns the number of rowns of this widget.
func (s *TasksWidget) RowCount() int {
	return len(s.filteredRows)
}

// Sort rotates to the next sort mode.
// SortByTaskService -> SortByTaskImage -> SortByTaskDesiredState -> SortByTaskState -> SortByTaskService
func (s *TasksWidget) Sort() {
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

// Unmount marks this widget as unmounted
func (s *TasksWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()

	s.mounted = false
	return nil

}

// Align aligns rows
func (s *TasksWidget) align() {
	x := s.screen.Bounds().Min.X
	width := s.screen.Bounds().Dx()

	s.tableTitle.SetX(x)
	s.tableTitle.SetWidth(width)

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, task := range s.totalRows {
		task.SetX(x)
		task.SetWidth(width)
	}

}

func (s *TasksWidget) filterRows() {

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

func (s *TasksWidget) calculateVisibleRows() {

	height := s.screen.Bounds().Dy() - widgetHeaderLength
	count := s.RowCount()

	//no screen
	if height < 0 || count == 0 {
		s.startIndex = 0
		s.endIndex = 0
		return
	}
	selected := s.selectedIndex
	//everything fits
	if count <= height {
		s.startIndex = 0
		s.endIndex = count
		return
	}
	//at the the start
	if selected == 0 {
		s.startIndex = 0
		s.endIndex = height
	} else if selected >= count-1 { //at the end
		s.startIndex = count - height
		s.endIndex = count
	} else if selected == s.endIndex { //scroll down by one
		s.startIndex++
		s.endIndex++
	} else if selected <= s.startIndex { //scroll up by one
		s.startIndex--
		s.endIndex--
	} else if selected > s.endIndex { // scroll
		s.startIndex = selected - height
		s.endIndex = selected
	}
}

// prepareForRendering sets the internal state of this widget so it is ready for
// rendering (i.e. Buffer()).
func (s *TasksWidget) prepareForRendering() {
	s.sortRows()
	s.filterRows()
	s.screen.Cursor().Max(s.RowCount() - 1)
	index := s.screen.Cursor().Position()
	if index < 0 {
		index = 0
	} else if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	s.selectedIndex = index
	s.calculateVisibleRows()
}

func (s *TasksWidget) updateHeader() {
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
		if header.Mode == appui.SortMode(sortMode) {
			c.Text = appui.DownArrow + colTitle
		} else {
			c.Text = colTitle
		}

	}

}

func (s *TasksWidget) visibleRows() []*TaskRow {
	return s.filteredRows[s.startIndex:s.endIndex]
}

func (s *TasksWidget) sortRows() {
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

var taskTableHeaders = []appui.SortableColumnHeader{
	{Title: "NAME", Mode: appui.SortMode(docker.SortByTaskService)},
	{Title: "IMAGE", Mode: appui.SortMode(docker.SortByTaskImage)},
	{Title: "NODE", Mode: appui.SortMode(docker.NoSortTask)},
	{Title: "DESIRED STATE", Mode: appui.SortMode(docker.SortByTaskDesiredState)},
	{Title: "CURRENT STATE", Mode: appui.SortMode(docker.SortByTaskState)},
	{Title: "ERROR", Mode: appui.SortMode(docker.NoSortTask)},
	{Title: "PORTS", Mode: appui.SortMode(docker.NoSortTask)},
}

func taskTableHeader() *termui.TableHeader {

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	header.AddColumn(taskTableHeaders[0].Title)
	header.AddColumn(taskTableHeaders[1].Title)
	header.AddColumn(taskTableHeaders[2].Title)
	header.AddFixedWidthColumn(taskTableHeaders[3].Title, 13)
	header.AddFixedWidthColumn(taskTableHeaders[4].Title, 22)
	header.AddColumn(taskTableHeaders[5].Title)
	header.AddColumn(taskTableHeaders[6].Title)

	return header
}

func createStackTableTitle() *termui.MarkupPar {
	p := termui.NewParFromMarkupText(appui.DryTheme, "")
	p.Bg = gizaktermui.Attribute(appui.DryTheme.Bg)
	p.TextBgColor = gizaktermui.Attribute(appui.DryTheme.Bg)
	p.TextFgColor = gizaktermui.Attribute(appui.DryTheme.Info)
	p.Border = false

	return p
}
