package appui

import (
	"sort"
	"strconv"
	"strings"
	"sync"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui/termui"
)

var defaultNetworkTableHeader = networkTableHeader()

var networkTableHeaders = []SortableColumnHeader{
	{`NETWORK ID`, SortMode(docker.SortNetworksByID)},
	{`NAME`, SortMode(docker.SortNetworksByName)},
	{`DRIVER`, SortMode(docker.SortNetworksByDriver)},
	{`CONTAINERS`, SortMode(docker.SortNetworksByContainerCount)},
	{`SERVICES`, SortMode(docker.SortNetworksByServiceCount)},
	{`SCOPE`, SortMode(docker.NoSortNetworks)},
	{`SUBNETS`, SortMode(docker.SortNetworksBySubnet)},
	{`GATEWAYS`, SortMode(docker.NoSortNetworks)},
}

// DockerNetworksWidget knows how render a container list
type DockerNetworksWidget struct {
	dockerDaemon         docker.NetworkAPI
	header               *termui.TableHeader
	filteredRows         []*NetworkRow
	totalRows            []*NetworkRow
	filterPattern        string
	screen               Screen
	selectedIndex        int
	startIndex, endIndex int
	sortMode             docker.SortMode

	sync.RWMutex
	mounted bool
}

// NewDockerNetworksWidget creates a renderer for a network list
func NewDockerNetworksWidget(dockerDaemon docker.NetworkAPI, s Screen) *DockerNetworksWidget {
	return &DockerNetworksWidget{
		dockerDaemon: dockerDaemon,
		header:       defaultNetworkTableHeader,
		sortMode:     docker.SortNetworksByID,
		screen:       s}
}

// Buffer returns the content of this widget as a termui.Buffer
func (s *DockerNetworksWidget) Buffer() gizaktermui.Buffer {
	s.RLock()
	defer s.RUnlock()
	buf := gizaktermui.NewBuffer()
	if !s.mounted {
		return buf
	}

	y := s.screen.Bounds().Min.Y

	s.prepareForRendering()
	widgetHeader := NewWidgetHeader()
	widgetHeader.HeaderEntry("Networks", strconv.Itoa(s.RowCount()))
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

	return buf
}

// Filter filters the network list by the given filter
func (s *DockerNetworksWidget) Filter(filter string) {
	s.Lock()
	defer s.Unlock()
	s.filterPattern = filter
}

// Mount tells this widget to be ready for rendering
func (s *DockerNetworksWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if s.mounted {
		return nil
	}
	networks, err := s.dockerDaemon.Networks()
	if err != nil {
		return err
	}

	networkRows := make([]*NetworkRow, len(networks))
	for i, network := range networks {
		networkRows[i] = NewNetworkRow(network, s.header)
	}
	s.totalRows = networkRows
	s.mounted = true
	s.align()

	return nil
}

// Name returns this widget name
func (s *DockerNetworksWidget) Name() string {
	return "DockerNetworksWidget"
}

// OnEvent runs the given command
func (s *DockerNetworksWidget) OnEvent(event EventCommand) error {
	if s.RowCount() > 0 {
		return event(s.filteredRows[s.selectedIndex].network.ID)
	}
	return nil
}

// RowCount returns the number of rows of this widget.
func (s *DockerNetworksWidget) RowCount() int {
	return len(s.filteredRows)

}

// Sort rotates to the next sort mode.
// SortNetworksByID -> SortNetworksByName -> SortNetworksByDriver
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

// Unmount tells this widget that it will not be rendering anymore
func (s *DockerNetworksWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()
	s.mounted = false
	return nil
}

// Align aligns rows
func (s *DockerNetworksWidget) align() {
	x := s.screen.Bounds().Min.X
	width := s.screen.Bounds().Dx()

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, network := range s.totalRows {
		network.SetX(x)
		network.SetWidth(width)
	}

}

func (s *DockerNetworksWidget) filterRows() {

	if s.filterPattern != "" {
		var rows []*NetworkRow

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

func (s *DockerNetworksWidget) calculateVisibleRows() {

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
func (s *DockerNetworksWidget) prepareForRendering() {
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
func (s *DockerNetworksWidget) updateHeader() {
	sortMode := SortMode(s.sortMode)

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

func (s *DockerNetworksWidget) sortRows() {
	rows := s.totalRows
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
	return s.filteredRows[s.startIndex:s.endIndex]
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
