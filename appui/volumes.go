package appui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/volume"
	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui/termui"
)

type volumesService interface {
	VolumeList(ctx context.Context) ([]*volume.Volume, error)
}

const (
	byDriver SortMode = iota + 1
	byName
)

var volumesTableHeaders = []SortableColumnHeader{
	{``, 0},
	{`DRIVER`, byDriver},
	{`VOLUME NAME`, byName},
}

// VolumesWidget shows information containers
type VolumesWidget struct {
	service              volumesService
	totalRows            []*VolumeRow
	filteredRows         []*VolumeRow
	header               *termui.TableHeader
	filterPattern        string
	selectedIndex        int
	startIndex, endIndex int
	sortBy               SortMode
	screen               Screen

	sync.RWMutex
	mounted bool
}

// NewVolumesWidget creates a VolumesWidget
func NewVolumesWidget(service volumesService, s Screen) *VolumesWidget {
	return &VolumesWidget{
		header:  volumesTableHeader(),
		service: service,
		screen:  s,
		sortBy:  byDriver}
}

// Buffer returns the content of this widget as a termui.Buffer
func (s *VolumesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	buf := gizaktermui.NewBuffer()

	if !s.mounted {
		return buf
	}
	s.prepareForRendering()
	y := s.screen.Bounds().Min.Y
	widgetHeader := NewWidgetHeader()
	widgetHeader.HeaderEntry("Volumes", strconv.Itoa(s.RowCount()))
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

	for i, volume := range s.visibleRows() {
		volume.SetY(y)
		y += volume.GetHeight()
		if i != selected {
			volume.NotHighlighted()
		} else {
			volume.Highlighted()
		}
		buf.Merge(volume.Buffer())
	}

	return buf
}

// Filter applies the given filter to the container list
func (s *VolumesWidget) Filter(filter string) {
	s.Lock()
	defer s.Unlock()
	s.filterPattern = filter

}

// Mount tells this widget to be ready for rendering
func (s *VolumesWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if s.mounted {
		s.align()
		return nil
	}
	s.mounted = true
	var rows []*VolumeRow
	vv, err := s.service.VolumeList(context.Background())
	if err != nil {
		return fmt.Errorf("could not retrieve volumes: %s", err.Error())
	}
	for _, v := range vv {
		rows = append(rows, NewVolumeRow(v, s.header))

	}
	s.totalRows = rows
	s.align()
	return nil
}

// Name returns this widget name
func (s *VolumesWidget) Name() string {
	return "VolumesWidget"
}

// OnEvent runs the given command
func (s *VolumesWidget) OnEvent(event EventCommand) error {
	if s.RowCount() <= 0 {
		return errors.New("The volume list is empty")
	} else if s.filteredRows[s.selectedIndex] == nil {
		return fmt.Errorf("The volume list does not have an element on pos %d", s.selectedIndex)
	}
	return event(s.filteredRows[s.selectedIndex].Name.Text)
}

// RowCount returns the number of rows of this widget.
func (s *VolumesWidget) RowCount() int {
	return len(s.filteredRows)
}

// Sort rotates to the next sort mode.
func (s *VolumesWidget) Sort() {
	s.Lock()
	defer s.Unlock()
	if s.sortBy == byName {
		s.sortBy = byDriver
	} else {
		s.sortBy++
	}
}

// Unmount this widget
func (s *VolumesWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()
	s.mounted = false
	return nil
}

// Align aligns rows
func (s *VolumesWidget) align() {
	x := s.screen.Bounds().Min.X
	width := s.screen.Bounds().Dx()

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, volume := range s.totalRows {
		volume.SetX(x)
		volume.SetWidth(width)
	}

}

func (s *VolumesWidget) filterRows() {

	if s.filterPattern != "" {
		var rows []*VolumeRow

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
func (s *VolumesWidget) prepareForRendering() {
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

func (s *VolumesWidget) updateTableHeader() {
	sortMode := s.sortBy

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header SortableColumnHeader
		if strings.Contains(colTitle, DownArrow) {
			colTitle = colTitle[DownArrowLength:]
		}
		for _, h := range volumesTableHeaders {
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

func (s *VolumesWidget) sortRows() {
	rows := s.totalRows
	mode := s.sortBy
	if mode == 0 {
		return
	}
	var sortFunc func(i, j int) bool

	switch mode {
	case byDriver:
		sortFunc = func(i, j int) bool {
			return rows[i].Driver.Text < rows[j].Driver.Text
		}
	case byName:
		sortFunc = func(i, j int) bool {
			return rows[i].Name.Text < rows[j].Name.Text
		}
	}
	sort.SliceStable(rows, sortFunc)
}

func (s *VolumesWidget) visibleRows() []*VolumeRow {
	return s.filteredRows[s.startIndex:s.endIndex]
}

func (s *VolumesWidget) calculateVisibleRows() {

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

func volumesTableHeader() *termui.TableHeader {

	header := termui.NewHeader(DryTheme)
	header.ColumnSpacing = DefaultColumnSpacing
	header.AddColumn(volumesTableHeaders[1].Title)
	header.AddColumn(volumesTableHeaders[2].Title)
	return header
}
