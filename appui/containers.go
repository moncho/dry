package appui

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"

	gizaktermui "github.com/gizak/termui"
)

var defaultContainerTableHeader = containerTableHeader()

var containerTableHeaders = []SortableColumnHeader{
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
	dockerDaemon         docker.ContainerAPI
	containers           []*ContainerRow
	showAllContainers    bool
	filters              []docker.ContainerFilter
	header               *termui.TableHeader
	sortMode             docker.SortMode
	filterPattern        string
	mounted              bool
	selectedIndex        int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewContainersWidget creates a ContainersWidget
func NewContainersWidget(dockerDaemon docker.ContainerAPI, y int) *ContainersWidget {
	w := ContainersWidget{
		dockerDaemon:      dockerDaemon,
		y:                 y,
		header:            defaultContainerTableHeader,
		height:            MainScreenAvailableHeight(),
		showAllContainers: false,
		sortMode:          docker.SortByContainerID,
		width:             ui.ActiveScreen.Dimensions.Width}

	RegisterWidget(docker.ContainerSource, &w)

	return &w

}

//Buffer returns the content of this widget as a termui.Buffer
func (s *ContainersWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	y := s.y
	buf := gizaktermui.NewBuffer()

	var filter string
	if s.filterPattern != "" {
		filter = fmt.Sprintf(
			"<b><blue> | Container name filter: </><yellow>%s</></> ", s.filterPattern)
	}

	widgetHeader := WidgetHeader("Containers", s.RowCount(), filter)
	widgetHeader.Y = y
	buf.Merge(widgetHeader.Buffer())
	y += widgetHeader.GetHeight()

	s.header.SetY(y)
	s.updateTableHeader()
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
func (s *ContainersWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if !s.mounted {

		var filters []docker.ContainerFilter
		if s.showAllContainers {
			filters = append(filters, docker.ContainerFilters.Unfiltered())
		} else {
			filters = append(filters, docker.ContainerFilters.Running())
		}
		dockerContainers := s.dockerDaemon.Containers(filters, s.sortMode)

		var rows []*ContainerRow
		for _, container := range dockerContainers {
			rows = append(rows, NewContainerRow(container, s.header))
		}
		s.containers = rows
		s.mounted = true
		s.align()
	}
	return nil
}

//Name returns this widget name
func (s *ContainersWidget) Name() string {
	return "ContainersWidget"
}

//OnEvent runs the given command
func (s *ContainersWidget) OnEvent(event EventCommand) error {
	if len(s.containers) > 0 {
		return event(s.containers[s.selectedIndex].container.ID)
	}
	return errors.New("The container list is empty")
}

//RowCount returns the number of rows of this widget.
func (s *ContainersWidget) RowCount() int {
	return len(s.containers)
}

//Sort rotates to the next sort mode.
//SortByContainerID -> SortByImage -> SortByStatus -> SortByName -> SortByContainerID
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
	s.mounted = false
}

//ToggleShowAllContainers toggles the show-all-containers state
func (s *ContainersWidget) ToggleShowAllContainers() {
	s.Lock()
	defer s.Unlock()

	s.showAllContainers = !s.showAllContainers
	s.mounted = false
}

//Unmount this widget
func (s *ContainersWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()
	s.mounted = false
	return nil
}

//Align aligns rows
func (s *ContainersWidget) align() {
	x := s.x
	width := s.width

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, container := range s.containers {
		container.SetX(x)
		container.SetWidth(width)
	}

}

func (s *ContainersWidget) highlightSelectedRow() {
	if s.RowCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	if s.selectedIndex < s.RowCount() {
		s.containers[s.selectedIndex].NotHighlighted()
	}
	s.selectedIndex = index
	s.containers[s.selectedIndex].Highlighted()
}

func (s *ContainersWidget) updateTableHeader() {
	sortMode := s.sortMode

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

/*func registerForDockerEvents(c *ContainersWidget) {
	docker.GlobalRegistry.Register(docker.ContainerSource,
		func(ctx context.Context, event events.Message) error {
			if event.Action == "start" || event.Action == "stop" {
				return c.Unmount()
			}
			return nil
		})

}*/
