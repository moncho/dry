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

var defaultNetworkTableHeader = networkTableHeader()

var networkTableHeaders = []networkHeaderColumn{
	{`NETWORK ID`, docker.SortNetworksByID},
	{`NAME`, docker.SortNetworksByName},
	{`DRIVER`, docker.SortNetworksByDriver},
	{`CONTAINERS`, docker.NoSortNetworks},
	{`SCOPE`, docker.NoSortNetworks},
}

//DockerNetworkRenderData holds information that might be
//used during image list rendering
type DockerNetworkRenderData struct {
	networks []types.NetworkResource
	sortMode docker.SortNetworksMode
}

//NewDockerNetworkRenderData creates render data structs
func NewDockerNetworkRenderData(networks []types.NetworkResource, sortMode docker.SortNetworksMode) *DockerNetworkRenderData {
	return &DockerNetworkRenderData{
		networks: networks,
		sortMode: sortMode,
	}
}

//DockerNetworksWidget knows how render a container list
type DockerNetworksWidget struct {
	networks             []*NetworkRow // List of columns.
	data                 *DockerNetworkRenderData
	header               *termui.TableHeader
	selectedIndex        int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewDockerNetworksWidget creates a renderer for a network list
func NewDockerNetworksWidget(y int) *DockerNetworksWidget {
	w := &DockerNetworksWidget{
		y:      y,
		header: defaultNetworkTableHeader,
		height: MainScreenAvailableHeight(),
		width:  ui.ActiveScreen.Dimensions.Width}

	return w
}

//PrepareToRender prepare this widget for rendering using the given data
func (s *DockerNetworksWidget) PrepareToRender(data *DockerNetworkRenderData) {
	s.Lock()
	defer s.Unlock()
	s.data = data
	var networks []*NetworkRow
	for _, network := range data.networks {
		networks = append(networks, NewNetworkRow(network, s.header))
	}
	s.networks = networks
	s.align()
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

//Buffer returns the content of this widget as a termui.Buffer
func (s *DockerNetworksWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	y := s.y

	buf := gizaktermui.NewBuffer()
	widgetHeader := WidgetHeader("Networks", s.RowCount(), "")
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

func (s *DockerNetworksWidget) updateHeader() {
	sortMode := s.data.sortMode

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header networkHeaderColumn
		if strings.Contains(colTitle, DownArrow) {
			colTitle = colTitle[DownArrowLength:]
		}
		for _, h := range networkTableHeaders {
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
func (s *DockerNetworksWidget) RowCount() int {
	return len(s.networks)
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
	s.networks[s.selectedIndex].Highlighted()
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

type networkHeaderColumn struct {
	title string // Title to display in the tableHeader.
	mode  docker.SortNetworksMode
}

func networkTableHeader() *termui.TableHeader {
	header := termui.NewHeader(DryTheme)
	header.ColumnSpacing = DefaultColumnSpacing
	header.AddColumn(networkTableHeaders[0].title)
	header.AddColumn(networkTableHeaders[1].title)
	header.AddFixedWidthColumn(networkTableHeaders[2].title, 12)
	header.AddFixedWidthColumn(networkTableHeaders[3].title, 12)
	header.AddColumn(networkTableHeaders[4].title)
	return header
}
