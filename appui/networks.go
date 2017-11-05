package appui

import (
	"sort"
	"strings"
	"sync"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

var defaultNetworkTableHeader = networkTableHeader()

var networkTableHeaders = []SortableColumnHeader{
	{`NETWORK ID`, docker.SortNetworksByID},
	{`NAME`, docker.SortNetworksByName},
	{`DRIVER`, docker.SortNetworksByDriver},
	{`CONTAINERS`, docker.SortNetworksByContainerCount},
	{`SERVICES`, docker.SortNetworksByServiceCount},
	{`SCOPE`, docker.NoSortNetworks},
	{`SUBNET`, docker.SortNetworksBySubnet},
	{`GATEWAY`, docker.NoSortNetworks},
}

//DockerNetworksWidget knows how render a container list
type DockerNetworksWidget struct {
	dockerDaemon         docker.NetworkAPI
	header               *termui.TableHeader
	networks             []*NetworkRow // List of columns.
	height, width        int
	selectedIndex        int
	startIndex, endIndex int
	x, y                 int
	sortMode             docker.SortMode
	mounted              bool
	sync.RWMutex
}

//NewDockerNetworksWidget creates a renderer for a network list
func NewDockerNetworksWidget(dockerDaemon docker.NetworkAPI, y int) *DockerNetworksWidget {
	w := DockerNetworksWidget{
		dockerDaemon: dockerDaemon,
		y:            y,
		header:       defaultNetworkTableHeader,
		height:       MainScreenAvailableHeight(),
		sortMode:     docker.SortNetworksByID,
		width:        ui.ActiveScreen.Dimensions.Width}

	RegisterWidget(docker.NetworkSource, &w)
	return &w
}

//Buffer returns the content of this widget as a termui.Buffer
func (s *DockerNetworksWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()

	buf := gizaktermui.NewBuffer()
	if s.mounted {
		y := s.y
		s.sortRows()
		s.updateHeader()

		widgetHeader := WidgetHeader("Networks", s.RowCount(), "")
		widgetHeader.Y = s.y
		buf.Merge(widgetHeader.Buffer())
		y += widgetHeader.GetHeight()

		s.header.SetY(y)
		buf.Merge(s.header.Buffer())

		y += s.header.GetHeight()

		s.highlightSelectedRow()
		for _, containerRow := range s.visibleRows() {
			containerRow.SetY(y)
			y += containerRow.GetHeight()
			buf.Merge(containerRow.Buffer())
		}
	}

	return buf
}

//Filter filters the network list by the given filter
func (s *DockerNetworksWidget) Filter(filter string) {

}

//Mount tells this widget to be ready for rendering
func (s *DockerNetworksWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if !s.mounted {
		networks, err := s.dockerDaemon.Networks()
		if err != nil {
			return err
		}

		networkRows := make([]*NetworkRow, len(networks))
		for i, network := range networks {
			networkRows[i] = NewNetworkRow(network, s.header)
		}
		s.networks = networkRows
		s.mounted = true
		s.align()
	}
	return nil
}

//Name returns this widget name
func (s *DockerNetworksWidget) Name() string {
	return "DockerNetworksWidget"
}

//OnEvent runs the given command
func (s *DockerNetworksWidget) OnEvent(event EventCommand) error {
	return event(s.networks[s.selectedIndex].network.ID)
}

//RowCount returns the number of rows of this widget.
func (s *DockerNetworksWidget) RowCount() int {
	return len(s.networks)
}

//Sort rotates to the next sort mode.
//SortNetworksByID -> SortNetworksByName -> SortNetworksByDriver
func (s *DockerNetworksWidget) Sort() {
	s.RLock()
	defer s.RUnlock()
	switch s.sortMode {
	case docker.SortNetworksByID:
		s.sortMode = docker.SortNetworksByName
	case docker.SortNetworksByName:
		s.sortMode = docker.SortNetworksByDriver
	case docker.SortNetworksByDriver:
		s.sortMode = docker.SortNetworksByContainerCount
	case docker.SortNetworksByContainerCount:
		s.sortMode = docker.SortNetworksByServiceCount
	case docker.SortNetworksByServiceCount:
		s.sortMode = docker.SortNetworksBySubnet
	case docker.SortNetworksBySubnet:
		s.sortMode = docker.SortNetworksByID
	}
}

//Unmount tells this widget that it will not be rendering anymore
func (s *DockerNetworksWidget) Unmount() error {
	s.RLock()
	defer s.RUnlock()
	s.mounted = false
	return nil
}

//Align aligns rows
func (s *DockerNetworksWidget) align() {
	x := s.x
	width := s.width

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, network := range s.networks {
		network.SetX(x)
		network.SetWidth(width)
	}

}

func (s *DockerNetworksWidget) updateHeader() {
	sortMode := s.sortMode

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header SortableColumnHeader
		if strings.Contains(colTitle, DownArrow) {
			colTitle = colTitle[DownArrowLength:]
		}
		for _, h := range networkTableHeaders {
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

func (s *DockerNetworksWidget) highlightSelectedRow() {
	if s.RowCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	s.selectedIndex = index
	for i, n := range s.networks {
		if i != index {
			n.NotHighlighted()
		} else {
			n.Highlighted()
		}
	}
}

func (s *DockerNetworksWidget) sortRows() {
	rows := s.networks
	mode := s.sortMode
	if s.sortMode == docker.NoSortNetworks {
		return
	}
	var sortAlg func(i, j int) bool

	switch mode {
	case docker.SortNetworksByID:
		sortAlg = func(i, j int) bool {
			return rows[i].ID.Text < rows[j].ID.Text
		}
	case docker.SortNetworksByName:
		sortAlg = func(i, j int) bool {
			return rows[i].Name.Text < rows[j].Name.Text
		}
	case docker.SortNetworksByDriver:
		sortAlg = func(i, j int) bool {
			return rows[i].Driver.Text < rows[j].Driver.Text
		}
	case docker.SortNetworksByContainerCount:
		sortAlg = func(i, j int) bool {
			return rows[i].Containers.Text < rows[j].Containers.Text
		}
	case docker.SortNetworksByServiceCount:
		sortAlg = func(i, j int) bool {
			return rows[i].Services.Text < rows[j].Services.Text
		}
	case docker.SortNetworksBySubnet:
		sortAlg = func(i, j int) bool {
			return rows[i].Subnet.Text < rows[j].Subnet.Text
		}

	}
	sort.SliceStable(rows, sortAlg)
}

func (s *DockerNetworksWidget) visibleRows() []*NetworkRow {

	//no screen
	if s.height < 0 {
		return nil
	}
	rows := s.networks
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

func networkTableHeader() *termui.TableHeader {
	header := termui.NewHeader(DryTheme)
	header.ColumnSpacing = DefaultColumnSpacing
	header.AddColumn(networkTableHeaders[0].Title)
	header.AddColumn(networkTableHeaders[1].Title)
	header.AddFixedWidthColumn(networkTableHeaders[2].Title, 12)
	header.AddFixedWidthColumn(networkTableHeaders[3].Title, 12)
	header.AddFixedWidthColumn(networkTableHeaders[4].Title, 12)
	header.AddColumn(networkTableHeaders[5].Title)
	header.AddColumn(networkTableHeaders[6].Title)
	header.AddColumn(networkTableHeaders[7].Title)

	return header
}
