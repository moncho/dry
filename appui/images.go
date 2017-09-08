package appui

import (
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
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

//DockerImageRenderData holds information that might be
//used during image list rendering
type DockerImageRenderData struct {
	images   []types.ImageSummary
	sortMode docker.SortImagesMode
}

//NewDockerImageRenderData creates render data structs
func NewDockerImageRenderData(images []types.ImageSummary, sortMode docker.SortImagesMode) *DockerImageRenderData {
	return &DockerImageRenderData{
		images:   images,
		sortMode: sortMode,
	}
}

//DockerImagesWidget knows how render a container list
type DockerImagesWidget struct {
	images               []*ImageRow // List of columns.
	data                 *DockerImageRenderData
	header               *termui.TableHeader
	selectedIndex        int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewDockerImagesWidget creates a renderer for a container list
func NewDockerImagesWidget(y int) *DockerImagesWidget {
	w := &DockerImagesWidget{
		y:      y,
		header: defaultImageTableHeader,
		height: MainScreenAvailableHeight(),
		width:  ui.ActiveScreen.Dimensions.Width}

	return w
}

//PrepareToRender prepare this widget for rendering using the given data
func (s *DockerImagesWidget) PrepareToRender(data *DockerImageRenderData) {
	s.Lock()
	defer s.Unlock()
	s.data = data
	var images []*ImageRow
	for _, image := range data.images {
		images = append(images, NewImageRow(image, s.header))
	}
	s.images = images
	s.align()
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

func (s *DockerImagesWidget) updateHeader() {
	sortMode := s.data.sortMode

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

//RowCount returns the number of rows of this widget.
func (s *DockerImagesWidget) RowCount() int {
	return len(s.images)
}
func (s *DockerImagesWidget) highlightSelectedRow() {
	if s.RowCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > s.RowCount() {
		index = s.RowCount() - 1
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
	mode  docker.SortImagesMode
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
