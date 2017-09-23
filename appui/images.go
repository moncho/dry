package appui

import (
	"strings"
	"sync"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

var defaultImageTableHeader = imageTableHeader()

var imageTableHeaders = []imageHeaderColumn{
	{`REPOSITORY`, docker.SortImagesByRepo},
	{`TAG`, docker.NoSortImages},
	{`ID`, docker.SortImagesByID},
	{`Created`, docker.SortImagesByCreationDate},
	{`Size`, docker.SortImagesBySize},
}

//DockerImagesWidget knows how render a container list
type DockerImagesWidget struct {
	images               []*ImageRow // List of columns.
	dockerDaemon         docker.ImageAPI
	mounted              bool
	header               *termui.TableHeader
	selectedIndex        int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sortMode             docker.SortMode
	sync.RWMutex
}

//NewDockerImagesWidget creates a renderer for a container list
func NewDockerImagesWidget(dockerDaemon docker.ImageAPI, y int) *DockerImagesWidget {
	w := DockerImagesWidget{
		y:            y,
		dockerDaemon: dockerDaemon,
		header:       defaultImageTableHeader,
		height:       MainScreenAvailableHeight(),
		sortMode:     docker.SortImagesByRepo,
		width:        ui.ActiveScreen.Dimensions.Width}

	RegisterWidget(docker.ImageSource, &w)

	return &w
}

//Buffer returns the content of this widget as a termui.Buffer
func (s *DockerImagesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	y := s.y

	buf := gizaktermui.NewBuffer()
	widgetHeader := WidgetHeader("Images", s.RowCount(), "")
	widgetHeader.Y = s.y
	buf.Merge(widgetHeader.Buffer())
	y += widgetHeader.GetHeight()

	s.header.SetY(y)
	s.updateHeader()
	buf.Merge(s.header.Buffer())

	y += s.header.GetHeight()

	s.highlightSelectedRow()
	for _, containerRow := range s.visibleRows() {
		containerRow.SetY(y)
		y += containerRow.GetHeight()
		buf.Merge(containerRow.Buffer())
	}

	return buf
}

//Mount tells this widget to be ready for rendering
func (s *DockerImagesWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if !s.mounted {
		images, err := s.dockerDaemon.Images()
		if err != nil {
			return err
		}

		docker.SortImages(images, s.sortMode)
		var imageRows []*ImageRow
		for _, image := range images {
			imageRows = append(imageRows, NewImageRow(image, s.header))
		}
		s.images = imageRows
		s.mounted = true
		s.align()
	}
	return nil
}

//Name returns this widget name
func (s *DockerImagesWidget) Name() string {
	return "DockerImagesWidget"
}

//OnEvent runs the given command
func (s *DockerImagesWidget) OnEvent(event EventCommand) error {
	return event(s.images[s.selectedIndex].image.ID)
}

//RowCount returns the number of rows of this widget.
func (s *DockerImagesWidget) RowCount() int {
	return len(s.images)
}

//SortImages rotates to the next sort mode.
//SortImagesByRepo -> SortImagesByID -> SortImagesByCreationDate -> SortImagesBySize -> SortImagesByRepo
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

//Unmount tells this widget that it will not be rendering anymore
func (s *DockerImagesWidget) Unmount() error {
	return nil
}

//Align aligns rows
func (s *DockerImagesWidget) align() {
	x := s.x
	width := s.width

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, image := range s.images {
		image.SetX(x)
		image.SetWidth(width)
	}

}

func (s *DockerImagesWidget) updateHeader() {
	sortMode := s.sortMode

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header imageHeaderColumn
		if strings.Contains(colTitle, DownArrow) {
			colTitle = colTitle[DownArrowLength:]
		}
		for _, h := range imageTableHeaders {
			if colTitle == h.title {
				header = h
				break
			}
		}
		if header.mode == sortMode {
			c.Text = DownArrow + colTitle
		} else {
			c.Text = colTitle
		}

	}

}

func (s *DockerImagesWidget) highlightSelectedRow() {
	if s.RowCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	if s.selectedIndex < s.RowCount() {
		s.images[s.selectedIndex].NotHighlighted()
	}
	s.selectedIndex = index
	s.images[s.selectedIndex].Highlighted()
}

func (s *DockerImagesWidget) visibleRows() []*ImageRow {

	//no screen
	if s.height < 0 {
		return nil
	}
	rows := s.images
	count := len(rows)
	selected := ui.ActiveScreen.Cursor.Position()
	//everything fits
	if count <= s.height {
		return rows
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
	return rows[s.startIndex:s.endIndex]
}

type imageHeaderColumn struct {
	title string // Title to display in the tableHeader.
	mode  docker.SortMode
}

func imageTableHeader() *termui.TableHeader {
	header := termui.NewHeader(DryTheme)
	header.ColumnSpacing = DefaultColumnSpacing
	header.AddColumn(imageTableHeaders[0].title)
	header.AddColumn(imageTableHeaders[1].title)
	header.AddFixedWidthColumn(imageTableHeaders[2].title, 12)
	header.AddFixedWidthColumn(imageTableHeaders[3].title, 12)
	header.AddColumn(imageTableHeaders[4].title)
	return header
}
