package swarm

import (
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui/termui"

	gizaktermui "github.com/gizak/termui"
)

var defaultStackTableHeader = stackTableHeader()

var stackTableHeaders = []appui.SortableColumnHeader{
	{Title: "NAME", Mode: appui.SortMode(docker.SortByStackName)},
	{Title: "SERVICES", Mode: appui.SortMode(docker.NoSortStack)},
	{Title: "ORCHESTRATOR", Mode: appui.SortMode(docker.NoSortStack)},
	{Title: "NETWORKS", Mode: appui.SortMode(docker.NoSortStack)},
	{Title: "CONFIGS", Mode: appui.SortMode(docker.NoSortStack)},
	{Title: "SECRETS", Mode: appui.SortMode(docker.NoSortStack)},
}

// StacksWidget shows information about services running on the Swarm
type StacksWidget struct {
	swarmClient          docker.SwarmAPI
	filteredRows         []*StackRow
	totalRows            []*StackRow
	filterPattern        string
	header               *termui.TableHeader
	selectedIndex        int
	offset               int
	screen               appui.Screen
	startIndex, endIndex int
	sortMode             docker.SortMode

	sync.RWMutex
	mounted bool
}

// NewStacksWidget creates a StacksWidget
func NewStacksWidget(swarmClient docker.SwarmAPI, s appui.Screen) *StacksWidget {
	return &StacksWidget{
		swarmClient:   swarmClient,
		header:        defaultStackTableHeader,
		selectedIndex: 0,
		offset:        0,
		screen:        s,
		sortMode:      docker.SortByServiceName}
}

// Buffer returns the content of this widget as a termui.Buffer
func (s *StacksWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()

	buf := gizaktermui.NewBuffer()
	if !s.mounted {
		return buf
	}
	y := s.screen.Bounds().Min.Y

	s.prepareForRendering()
	widgetHeader := appui.NewWidgetHeader()
	widgetHeader.HeaderEntry("Stacks", strconv.Itoa(s.RowCount()))
	if s.filterPattern != "" {
		widgetHeader.HeaderEntry("Active filter", s.filterPattern)
	}
	widgetHeader.Y = y
	buf.Merge(widgetHeader.Buffer())
	y += widgetHeader.GetHeight()
	//Empty line between the header and the rest of the content
	y++
	s.updateHeader()
	s.header.SetY(y)
	buf.Merge(s.header.Buffer())
	y += s.header.GetHeight()

	selected := s.selectedIndex - s.startIndex

	for i, stackRow := range s.visibleRows() {
		stackRow.SetY(y)
		y += stackRow.GetHeight()
		if i != selected {
			stackRow.NotHighlighted()
		} else {
			stackRow.Highlighted()
		}
		buf.Merge(stackRow.Buffer())
	}
	return buf
}

// Filter applies the given filter to the container list
func (s *StacksWidget) Filter(filter string) {
	s.Lock()
	defer s.Unlock()
	s.filterPattern = filter
}

// Mount prepares this widget for rendering
func (s *StacksWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if s.mounted {
		s.align()
		return nil
	}
	s.mounted = true
	var rows []*StackRow
	if stacks, err := s.swarmClient.Stacks(); err == nil {
		for _, stack := range stacks {
			rows = append(rows, NewStackRow(stack, s.header))
		}
		s.totalRows = rows
	} else {
		return err
	}
	s.align()
	return nil

}

// Name returns this widget name
func (s *StacksWidget) Name() string {
	return "StacksWidget"
}

// RowCount returns the number of rowns of this widget.
func (s *StacksWidget) RowCount() int {
	return len(s.filteredRows)
}

// OnEvent runs the given command
func (s *StacksWidget) OnEvent(event appui.EventCommand) error {
	if s.RowCount() > 0 {
		return event(s.filteredRows[s.selectedIndex].stack.Name)
	}
	return nil
}

// Sort rotates to the next sort mode.
// SortByServiceName -> SortByServiceImage -> SortByServiceName
func (s *StacksWidget) Sort() {
	//There is one sort mode
}

// Unmount marks this widget as unmounted
func (s *StacksWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()

	s.mounted = false
	return nil

}

// Align aligns rows
func (s *StacksWidget) align() {
	x := s.screen.Bounds().Min.X
	width := s.screen.Bounds().Dx()

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, service := range s.totalRows {
		service.SetX(x)
		service.SetWidth(width)
	}

}

func (s *StacksWidget) filterRows() {

	if s.filterPattern != "" {
		var rows []*StackRow

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

func (s *StacksWidget) calculateVisibleRows() {

	count := s.RowCount()
	height := s.screen.Bounds().Dy() - widgetHeaderLength

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
func (s *StacksWidget) prepareForRendering() {
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

func (s *StacksWidget) updateHeader() {
	sortMode := s.sortMode

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header appui.SortableColumnHeader
		if strings.Contains(colTitle, appui.DownArrow) {
			colTitle = colTitle[appui.DownArrowLength:]
		}
		for _, h := range stackTableHeaders {
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

func (s *StacksWidget) visibleRows() []*StackRow {
	return s.filteredRows[s.startIndex:s.endIndex]
}

func (s *StacksWidget) sortRows() {
	rows := s.totalRows
	mode := s.sortMode
	if s.sortMode == docker.NoSortService {
		return
	}
	var sortAlg func(i, j int) bool
	switch mode {
	case docker.SortByStackName:
		sortAlg = func(i, j int) bool {
			return rows[i].Name.Text < rows[j].Name.Text
		}
	}
	sort.SliceStable(rows, sortAlg)
}

func stackTableHeader() *termui.TableHeader {

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	for _, t := range stackTableHeaders {
		header.AddColumn(t.Title)
	}

	return header
}
