package appui

import (
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/image"
	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui/termui"
)

var defaultImageTableHeader = imageTableHeader()

var imageTableHeaders = []SortableColumnHeader{
	{`REPOSITORY`, SortMode(docker.SortImagesByRepo)},
	{`TAG`, SortMode(docker.NoSortImages)},
	{`ID`, SortMode(docker.SortImagesByID)},
	{`Created`, SortMode(docker.SortImagesByCreationDate)},
	{`Size`, SortMode(docker.SortImagesBySize)},
}

// DockerImagesWidget knows how render a container list
type DockerImagesWidget struct {
	images               func() ([]image.Summary, error)
	filteredRows         []*ImageRow
	totalRows            []*ImageRow
	filterPattern        string
	header               *termui.TableHeader
	selectedIndex        int
	startIndex, endIndex int
	sortMode             docker.SortMode
	screen               Screen

	sync.RWMutex
	mounted bool
}

// NewDockerImagesWidget creates a widget to show Docker images.
func NewDockerImagesWidget(images func() ([]image.Summary, error), s Screen) *DockerImagesWidget {
	return &DockerImagesWidget{
		images:   images,
		header:   defaultImageTableHeader,
		screen:   s,
		sortMode: docker.SortImagesByRepo}
}

// Buffer returns the content of this widget as a termui.Buffer
func (s *DockerImagesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	y := s.screen.Bounds().Min.Y
	buf := gizaktermui.NewBuffer()
	if s.mounted {
		s.prepareForRendering()
		widgetHeader := NewWidgetHeader()
		widgetHeader.HeaderEntry("Images", strconv.Itoa(s.RowCount()))
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

		for i, imageRow := range s.visibleRows() {
			imageRow.SetY(y)
			y += imageRow.GetHeight()
			if i != selected {
				imageRow.NotHighlighted()
			} else {
				imageRow.Highlighted()
			}
			buf.Merge(imageRow.Buffer())
		}
	}
	return buf
}

// Filter filters the image list by the given filter
func (s *DockerImagesWidget) Filter(filter string) {
	s.Lock()
	defer s.Unlock()
	s.filterPattern = filter
}

// Mount tells this widget to be ready for rendering
func (s *DockerImagesWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if s.mounted {
		return nil
	}
	images, err := s.images()
	if err != nil {
		return err
	}

	imageRows := make([]*ImageRow, len(images))
	for i, image := range images {
		imageRows[i] = NewImageRow(image, s.header)
	}
	s.totalRows = imageRows
	s.mounted = true
	s.align()

	return nil
}

// Name returns this widget name
func (s *DockerImagesWidget) Name() string {
	return "DockerImagesWidget"
}

// OnEvent runs the given command
func (s *DockerImagesWidget) OnEvent(event EventCommand) error {
	if s.RowCount() > 0 {
		return event(s.filteredRows[s.selectedIndex].image.ID)
	}
	return nil
}

// RowCount returns the number of rows of this widget.
func (s *DockerImagesWidget) RowCount() int {
	return len(s.filteredRows)
}

// Sort rotates to the next sort mode.
// SortImagesByRepo -> SortImagesByID -> SortImagesByCreationDate -> SortImagesBySize -> SortImagesByRepo
func (s *DockerImagesWidget) Sort() {
	s.RLock()
	defer s.RUnlock()
	switch s.sortMode {
	case docker.SortImagesByRepo:
		s.sortMode = docker.SortImagesByID
	case docker.SortImagesByID:
		s.sortMode = docker.SortImagesByCreationDate
	case docker.SortImagesByCreationDate:
		s.sortMode = docker.SortImagesBySize
	case docker.SortImagesBySize:
		s.sortMode = docker.SortImagesByRepo
	}
	s.mounted = false
}

// Unmount tells this widget that it will not be rendering anymore
func (s *DockerImagesWidget) Unmount() error {
	s.RLock()
	defer s.RUnlock()
	s.mounted = false
	return nil
}

// Align aligns rows
func (s *DockerImagesWidget) align() {
	x := s.screen.Bounds().Min.X
	width := s.screen.Bounds().Dx()

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, image := range s.totalRows {
		image.SetX(x)
		image.SetWidth(width)
	}

}

func (s *DockerImagesWidget) filterRows() {

	if s.filterPattern != "" {
		var rows []*ImageRow

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

func (s *DockerImagesWidget) calculateVisibleRows() {

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
func (s *DockerImagesWidget) prepareForRendering() {
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

func (s *DockerImagesWidget) updateHeader() {
	sortMode := SortMode(s.sortMode)

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header SortableColumnHeader
		if strings.Contains(colTitle, DownArrow) {
			colTitle = colTitle[DownArrowLength:]
		}
		for _, h := range imageTableHeaders {
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

func (s *DockerImagesWidget) sortRows() {
	rows := s.totalRows
	mode := s.sortMode
	if s.sortMode == docker.NoSortImages {
		return
	}
	var sortAlg func(i, j int) bool

	switch mode {
	case docker.SortImagesByRepo:
		sortAlg = func(i, j int) bool {
			if rows[i].Repository.Text != rows[j].Repository.Text {
				return rows[i].Repository.Text < rows[j].Repository.Text
			}
			return rows[i].Tag.Text < rows[j].Tag.Text
		}
	case docker.SortImagesByID:
		sortAlg = func(i, j int) bool {
			return rows[i].ID.Text < rows[j].ID.Text
		}
	case docker.SortImagesByCreationDate:
		sortAlg = func(i, j int) bool {
			return rows[i].CreatedSinceValue > rows[j].CreatedSinceValue
		}
	case docker.SortImagesBySize:
		sortAlg = func(i, j int) bool {
			return rows[i].SizeValue < rows[j].SizeValue
		}

	}
	sort.SliceStable(rows, sortAlg)
}

func (s *DockerImagesWidget) visibleRows() []*ImageRow {
	return s.filteredRows[s.startIndex:s.endIndex]
}

func imageTableHeader() *termui.TableHeader {
	header := termui.NewHeader(DryTheme)
	header.ColumnSpacing = DefaultColumnSpacing
	header.AddColumn(imageTableHeaders[0].Title)
	header.AddColumn(imageTableHeaders[1].Title)
	header.AddFixedWidthColumn(imageTableHeaders[2].Title, 12)
	header.AddFixedWidthColumn(imageTableHeaders[3].Title, 12)
	header.AddColumn(imageTableHeaders[4].Title)
	return header
}
