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
func NewContainersWidget(data *DockerPsRenderData, y int) *ContainersWidget {
	w := &ContainersWidget{
		data:          data,
		header:        defaultContainerTableHeader,
		selectedIndex: 0,
		offset:        0,
		x:             0,
		y:             y,
		height:        MainScreenAvailableHeight(),
		width:         ui.ActiveScreen.Dimensions.Width}
	for _, container := range data.containers {
		w.containers = append(w.containers, NewContainerRow(container, w.header))
	}
	w.align()

	return w

}

//Align aligns rows
func (s *ContainersWidget) align() {
	y := s.y
	x := s.x
	width := s.width

	s.header.SetWidth(width)
	s.header.SetY(y)
	s.header.SetX(x)

	for _, service := range s.containers {
		service.SetX(x)
		service.SetWidth(width)
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
	for _, service := range s.visibleRows() {
		service.SetY(y)
		service.Height = 1
		y += service.GetHeight()
		buf.Merge(service.Buffer())
	}

	return buf
}

//RowCount returns the number of rowns of this widget.
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
	s.containers[s.selectedIndex].NotHighlighted()
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
			colTitle = colTitle[DownArrowLenght:]
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
