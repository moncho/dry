package appui

import (
	"strings"
	"sync"

	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"

	gizaktermui "github.com/gizak/termui"
)

var defaultContainerTableHeader = containerTableHeader()

var containerTableHeaders = []headerColumn{
	{``, docker.NoSort},
	{`CONTAINER`, docker.SortByContainerID},
	{`IMAGE`, docker.SortByImage},
	{`COMMAND`, docker.NoSort},
	{`STATUS`, docker.SortByStatus},
	{`PORTS`, docker.NoSort},
	{`NAMES`, docker.SortByName},
}

//ContainersWidget shows information containers
type ContainersWidget struct {
	data                 *DockerPsRenderData
	containers           []*ContainerRow
	header               *termui.TableHeader
	selectedIndex        int
	offset               int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewContainersWidget creates a ContainersWidget
func NewContainersWidget(y int) *ContainersWidget {
	w := &ContainersWidget{
		y:      y,
		header: defaultContainerTableHeader,
		height: MainScreenAvailableHeight(),
		width:  ui.ActiveScreen.Dimensions.Width}

	return w

}

//PrepareToRender prepares this widget for rendering
func (s *ContainersWidget) PrepareToRender(data *DockerPsRenderData) {
	s.Lock()
	defer s.Unlock()
	s.data = data
	var containers []*ContainerRow
	for _, container := range data.containers {
		containers = append(containers, NewContainerRow(container, s.header))
	}
	s.containers = containers
	s.align()
}

//Align aligns rows
func (s *ContainersWidget) align() {
	y := s.y
	x := s.x
	width := s.width

	s.header.SetWidth(width)
	s.header.SetY(y)
	s.header.SetX(x)

	for _, container := range s.containers {
		container.SetX(x)
		container.SetWidth(width)
	}

}

//Buffer returns the content of this widget as a termui.Buffer
func (s *ContainersWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()

	buf := gizaktermui.NewBuffer()
	s.updateHeader()

	buf.Merge(s.header.Buffer())

	y := s.y
	y += s.header.GetHeight()

	s.highlightSelectedRow()
	for _, containerRow := range s.visibleRows() {
		containerRow.SetY(y)
		y += containerRow.GetHeight()
		buf.Merge(containerRow.Buffer())
	}

	return buf
}

//RowCount returns the number of rows of this widget.
func (s *ContainersWidget) RowCount() int {
	return len(s.containers)
}
func (s *ContainersWidget) highlightSelectedRow() {
	if s.RowCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	s.selectedIndex = index
	s.containers[s.selectedIndex].Highlighted()
}

//OnEvent runs the given command
func (s *ContainersWidget) OnEvent(event EventCommand) error {
	return event(s.containers[s.selectedIndex].container.ID)
}

func (s *ContainersWidget) updateHeader() {
	sortMode := s.data.sortMode

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header headerColumn
		if strings.Contains(colTitle, DownArrow) {
			colTitle = colTitle[DownArrowLength:]
		}
		for _, h := range containerTableHeaders {
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

func (s *ContainersWidget) visibleRows() []*ContainerRow {

	//no screen
	if s.height < 0 {
		return nil
	}
	rows := s.containers
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

type headerColumn struct {
	title string // Title to display in the tableHeader.
	mode  docker.SortMode
}

func containerTableHeader() *termui.TableHeader {

	header := termui.NewHeader(DryTheme)
	header.ColumnSpacing = DefaultColumnSpacing
	header.AddFixedWidthColumn(containerTableHeaders[0].title, 2)
	header.AddFixedWidthColumn(containerTableHeaders[1].title, 12)
	header.AddColumn(containerTableHeaders[2].title)
	header.AddColumn(containerTableHeaders[3].title)
	header.AddFixedWidthColumn(containerTableHeaders[4].title, 18)
	header.AddColumn(containerTableHeaders[5].title)
	header.AddColumn(containerTableHeaders[6].title)

	return header
}
