package appui

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui/termui"

	gizaktermui "github.com/gizak/termui"
)

// the default length of a widget header
const widgetHeaderLength = 4

var defaultContainerTableHeader = containerTableHeader()

var containerTableHeaders = []SortableColumnHeader{
	{``, SortMode(docker.NoSort)},
	{`CONTAINER`, SortMode(docker.SortByContainerID)},
	{`IMAGE`, SortMode(docker.SortByImage)},
	{`COMMAND`, SortMode(docker.NoSort)},
	{`STATUS`, SortMode(docker.SortByStatus)},
	{`PORTS`, SortMode(docker.NoSort)},
	{`NAMES`, SortMode(docker.SortByName)},
}

// ContainersWidget shows information containers
type ContainersWidget struct {
	dockerDaemon         docker.ContainerAPI
	totalRows            []*ContainerRow
	filteredRows         []*ContainerRow
	header               *termui.TableHeader
	filterPattern        string
	selectedIndex        int
	startIndex, endIndex int
	sortMode             docker.SortMode
	screen               Screen
	showAllContainers    bool

	sync.RWMutex
	mounted bool
}

// NewContainersWidget creates a ContainersWidget
func NewContainersWidget(dockerDaemon docker.ContainerAPI, s Screen) *ContainersWidget {
	return &ContainersWidget{
		dockerDaemon:      dockerDaemon,
		header:            defaultContainerTableHeader,
		screen:            s,
		showAllContainers: false,
		sortMode:          docker.SortByContainerID}
}

// Buffer returns the content of this widget as a termui.Buffer
func (s *ContainersWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	buf := gizaktermui.NewBuffer()

	if s.mounted {
		s.prepareForRendering()
		y := s.screen.Bounds().Min.Y
		widgetHeader := NewWidgetHeader()
		widgetHeader.HeaderEntry("Containers", strconv.Itoa(s.RowCount()))
		if s.filterPattern != "" {
			widgetHeader.HeaderEntry("Active filter", s.filterPattern)
		}
		widgetHeader.Buffer()
		widgetHeader.Y = y
		buf.Merge(widgetHeader.Buffer())
		y += widgetHeader.GetHeight()
		//Empty line between the header and the rest of the content
		y++
		s.header.SetY(y)
		s.updateTableHeader()
		buf.Merge(s.header.Buffer())

		y += s.header.GetHeight()

		selected := s.selectedIndex - s.startIndex

		for i, containerRow := range s.visibleRows() {
			containerRow.SetY(y)
			y += containerRow.GetHeight()
			if i != selected {
				containerRow.NotHighlighted()
			} else {
				containerRow.Highlighted()
			}
			buf.Merge(containerRow.Buffer())
		}
	}
	return buf
}

// Filter applies the given filter to the container list
func (s *ContainersWidget) Filter(filter string) {
	s.Lock()
	defer s.Unlock()
	s.filterPattern = filter

}

// Mount tells this widget to be ready for rendering
func (s *ContainersWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if s.mounted {
		return nil
	}

	var filters []docker.ContainerFilter
	if s.showAllContainers {
		filters = append(filters, docker.ContainerFilters.Unfiltered())
	} else {
		filters = append(filters, docker.ContainerFilters.Running())
	}
	dockerContainers := s.dockerDaemon.Containers(filters, s.sortMode)

	rows := make([]*ContainerRow, len(dockerContainers))
	for i, container := range dockerContainers {
		rows[i] = NewContainerRow(container, s.header)
	}
	s.totalRows = rows
	s.mounted = true
	s.align()

	return nil
}

// Name returns this widget name
func (s *ContainersWidget) Name() string {
	return "ContainersWidget"
}

// OnEvent runs the given command
func (s *ContainersWidget) OnEvent(event EventCommand) error {
	if s.RowCount() <= 0 {
		return errors.New("The container list is empty")
	} else if s.filteredRows[s.selectedIndex] == nil {
		return fmt.Errorf("The container list does not have an element on pos %d", s.selectedIndex)
	}
	return event(s.filteredRows[s.selectedIndex].container.ID)
}

// RowCount returns the number of rows of this widget.
func (s *ContainersWidget) RowCount() int {
	return len(s.filteredRows)
}

// Sort rotates to the next sort mode.
// SortByContainerID -> SortByImage -> SortByStatus -> SortByName -> SortByContainerID
func (s *ContainersWidget) Sort() {
	s.Lock()
	defer s.Unlock()
	switch s.sortMode {
	case docker.SortByContainerID:
		s.sortMode = docker.SortByImage
	case docker.SortByImage:
		s.sortMode = docker.SortByStatus
	case docker.SortByStatus:
		s.sortMode = docker.SortByName
	case docker.SortByName:
		s.sortMode = docker.SortByContainerID
	default:
	}
}

// ToggleShowAllContainers toggles the show-all-containers state
func (s *ContainersWidget) ToggleShowAllContainers() {
	s.Lock()
	defer s.Unlock()

	s.showAllContainers = !s.showAllContainers
	s.mounted = false
}

// Unmount this widget
func (s *ContainersWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()
	s.mounted = false
	return nil
}

// Align aligns rows
func (s *ContainersWidget) align() {
	x := s.screen.Bounds().Min.X
	width := s.screen.Bounds().Dx()

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, container := range s.totalRows {
		container.SetX(x)
		container.SetWidth(width)
	}

}

func (s *ContainersWidget) filterRows() {

	if s.filterPattern != "" {
		var rows []*ContainerRow

		for _, row := range s.totalRows {
			if RowFilters.ByPattern(s.filterPattern)(row) {
				rows = append(rows, row)
			}
		}
		s.filteredRows = rows
	} else {
		s.filteredRows = s.totalRows
	}
}

// prepareForRendering sets the internal state of this widget so it is ready for
// rendering(i.e. Buffer()).
func (s *ContainersWidget) prepareForRendering() {
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

func (s *ContainersWidget) updateTableHeader() {
	sortMode := SortMode(s.sortMode)

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header SortableColumnHeader
		if strings.Contains(colTitle, DownArrow) {
			colTitle = colTitle[DownArrowLength:]
		}
		for _, h := range containerTableHeaders {
			if colTitle == h.Title {
				header = h
				break
			}
		}
		if header.Mode == sortMode {
			c.Text = DownArrow + colTitle
		} else {
			c.Text = colTitle
		}

	}

}

func (s *ContainersWidget) sortRows() {
	rows := s.totalRows
	mode := s.sortMode
	if s.sortMode == docker.NoSort {
		return
	}
	var sortAlg func(i, j int) bool

	switch mode {
	case docker.SortByContainerID:
		sortAlg = func(i, j int) bool {
			return rows[i].ID.Text < rows[j].ID.Text
		}
	case docker.SortByImage:
		sortAlg = func(i, j int) bool {
			return rows[i].Image.Text < rows[j].Image.Text
		}
	case docker.SortByStatus:
		sortAlg = func(i, j int) bool {
			return rows[i].Status.Text < rows[j].Status.Text
		}
	case docker.SortByName:
		sortAlg = func(i, j int) bool {
			return rows[i].Names.Text < rows[j].Names.Text
		}

	}
	sort.SliceStable(rows, sortAlg)
}

func (s *ContainersWidget) visibleRows() []*ContainerRow {
	return s.filteredRows[s.startIndex:s.endIndex]
}

func (s *ContainersWidget) calculateVisibleRows() {

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

func containerTableHeader() *termui.TableHeader {

	header := termui.NewHeader(DryTheme)
	header.ColumnSpacing = DefaultColumnSpacing
	header.AddFixedWidthColumn(containerTableHeaders[0].Title, 2)
	header.AddFixedWidthColumn(containerTableHeaders[1].Title, 12)
	header.AddColumn(containerTableHeaders[2].Title)
	header.AddColumn(containerTableHeaders[3].Title)
	header.AddFixedWidthColumn(containerTableHeaders[4].Title, 18)
	header.AddColumn(containerTableHeaders[5].Title)
	header.AddColumn(containerTableHeaders[6].Title)

	return header
}
