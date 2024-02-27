package swarm

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui/termui"

	gizaktermui "github.com/gizak/termui"
)

var defaultServiceTableHeader = serviceTableHeader()

var serviceTableHeaders = []appui.SortableColumnHeader{
	{Title: "NAME", Mode: appui.SortMode(docker.SortByServiceName)},
	{Title: "MODE", Mode: appui.SortMode(docker.NoSortService)},
	{Title: "REPLICAS", Mode: appui.SortMode(docker.NoSortService)},
	{Title: "SERVICE PORT(S)", Mode: appui.SortMode(docker.NoSortService)},
	{Title: "IMAGE", Mode: appui.SortMode(docker.SortByServiceImage)},
}

// ServicesWidget shows information about services running on the Swarm
type ServicesWidget struct {
	filteredRows         []*ServiceRow
	filterPattern        string
	header               *termui.TableHeader
	screen               appui.Screen
	selectedIndex        int
	startIndex, endIndex int
	sortMode             docker.SortMode
	swarmClient          docker.SwarmAPI
	totalRows            []*ServiceRow

	sync.RWMutex
	mounted bool
}

// NewServicesWidget creates a ServicesWidget
func NewServicesWidget(swarmClient docker.SwarmAPI, s appui.Screen) *ServicesWidget {
	return &ServicesWidget{
		swarmClient:   swarmClient,
		header:        defaultServiceTableHeader,
		selectedIndex: 0,
		screen:        s,
		sortMode:      docker.SortByServiceName}

}

// Buffer returns the content of this widget as a termui.Buffer
func (s *ServicesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	buf := gizaktermui.NewBuffer()
	if !s.mounted {
		return buf
	}
	y := s.screen.Bounds().Min.Y

	s.prepareForRendering()
	widgetHeader := appui.NewWidgetHeader()
	widgetHeader.HeaderEntry("Services", strconv.Itoa(s.RowCount()))
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

	for i, serviceRow := range s.visibleRows() {
		serviceRow.SetY(y)
		y += serviceRow.GetHeight()
		if i != selected {
			serviceRow.NotHighlighted()
		} else {
			serviceRow.Highlighted()
		}
		buf.Merge(serviceRow.Buffer())
	}
	return buf
}

// Filter applies the given filter to the container list
func (s *ServicesWidget) Filter(filter string) {
	s.Lock()
	defer s.Unlock()
	s.filterPattern = filter
}

// Mount prepares this widget for rendering
func (s *ServicesWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if s.mounted {
		s.align()
		return nil
	}
	s.mounted = true
	var rows []*ServiceRow
	if services, servicesInfo, err := getServiceInfo(s.swarmClient); err == nil {
		for _, service := range services {
			rows = append(rows, NewServiceRow(service, servicesInfo[service.ID], s.header))
		}
	}
	s.totalRows = rows
	s.align()
	return nil
}

// Name returns this widget name
func (s *ServicesWidget) Name() string {
	return "ServicesWidget"
}

// OnEvent runs the given command
func (s *ServicesWidget) OnEvent(event appui.EventCommand) error {
	if s.RowCount() > 0 {
		return event(s.filteredRows[s.selectedIndex].service.ID)
	}
	return nil
}

// RowCount returns the number of rowns of this widget.
func (s *ServicesWidget) RowCount() int {
	return len(s.filteredRows)
}

// Sort rotates to the next sort mode.
// SortByServiceName -> SortByServiceImage -> SortByServiceName
func (s *ServicesWidget) Sort() {
	s.Lock()
	defer s.Unlock()
	switch s.sortMode {
	case docker.SortByServiceName:
		s.sortMode = docker.SortByServiceImage
	case docker.SortByServiceImage:
		s.sortMode = docker.SortByServiceName
	}
}

// Unmount marks this widget as unmounted
func (s *ServicesWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()

	s.mounted = false
	return nil

}

// Align aligns rows
func (s *ServicesWidget) align() {
	x := s.screen.Bounds().Min.X
	width := s.screen.Bounds().Dx()

	s.header.SetWidth(width)
	s.header.SetX(x)

	for _, service := range s.totalRows {
		service.SetX(x)
		service.SetWidth(width)
	}

}

func (s *ServicesWidget) filterRows() {

	if s.filterPattern != "" {
		var rows []*ServiceRow

		for _, row := range s.totalRows {
			if appui.RowFilters.ByPattern(s.filterPattern)(row) {
				rows = append(rows, row)
			}
		}
		s.filteredRows = rows
	} else {
		s.filteredRows = s.totalRows
	}
}

func (s *ServicesWidget) calculateVisibleRows() {

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
func (s *ServicesWidget) prepareForRendering() {
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

func (s *ServicesWidget) updateHeader() {
	sortMode := s.sortMode

	for _, c := range s.header.Columns {
		colTitle := c.Text
		var header appui.SortableColumnHeader
		if strings.Contains(colTitle, appui.DownArrow) {
			colTitle = colTitle[appui.DownArrowLength:]
		}
		for _, h := range serviceTableHeaders {
			if colTitle == h.Title {
				header = h
				break
			}
		}
		if header.Mode == appui.SortMode(sortMode) {
			c.Text = appui.DownArrow + colTitle
		} else {
			c.Text = colTitle
		}

	}

}

func (s *ServicesWidget) visibleRows() []*ServiceRow {
	return s.filteredRows[s.startIndex:s.endIndex]
}

func (s *ServicesWidget) sortRows() {
	rows := s.totalRows
	mode := s.sortMode
	if s.sortMode == docker.NoSortService {
		return
	}
	var sortAlg func(i, j int) bool
	switch mode {
	case docker.SortByServiceImage:
		sortAlg = func(i, j int) bool {
			return rows[i].Image.Text < rows[j].Image.Text
		}
	case docker.SortByServiceName:
		sortAlg = func(i, j int) bool {
			return rows[i].Name.Text < rows[j].Name.Text
		}

	}
	sort.SliceStable(rows, sortAlg)
}

func serviceTableHeader() *termui.TableHeader {

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	header.AddFixedWidthColumn(serviceTableHeaders[0].Title, 30)
	header.AddFixedWidthColumn(serviceTableHeaders[1].Title, 12)
	header.AddFixedWidthColumn(serviceTableHeaders[2].Title, 10)
	header.AddColumn(serviceTableHeaders[3].Title)
	header.AddColumn(serviceTableHeaders[4].Title)

	return header
}

func getServiceInfo(swarmClient docker.SwarmAPI) ([]swarm.Service, map[string]ServiceListInfo, error) {

	serviceFilters := filters.NewArgs()
	serviceFilters.Add("runtime", string(swarm.RuntimeContainer))
	services, err := swarmClient.Services()
	if err != nil {
		return nil, nil, err
	}

	info := map[string]ServiceListInfo{}
	if len(services) > 0 {

		tasks, err := swarmClient.ServiceTasks(serviceIDs(services)...)
		if err != nil {
			return nil, nil, err
		}

		nodes, err := swarmClient.Nodes()
		if err != nil {
			return nil, nil, err
		}

		info = getServicesStatus(services, nodes, tasks)
	}
	return services, info, nil
}

func serviceIDs(services []swarm.Service) []string {

	ids := make([]string, len(services))
	for i, service := range services {
		ids[i] = service.ID
	}

	return ids
}

// getServicesStatus returns a map of mode and replicas
func getServicesStatus(services []swarm.Service, nodes []swarm.Node, tasks []swarm.Task) map[string]ServiceListInfo {
	running := map[string]int{}
	tasksNoShutdown := map[string]int{}

	activeNodes := make(map[string]struct{})
	for _, n := range nodes {
		if n.Status.State != swarm.NodeStateDown {
			activeNodes[n.ID] = struct{}{}
		}
	}

	for _, task := range tasks {
		if task.DesiredState != swarm.TaskStateShutdown {
			tasksNoShutdown[task.ServiceID]++
		}

		if _, nodeActive := activeNodes[task.NodeID]; nodeActive && task.Status.State == swarm.TaskStateRunning {
			running[task.ServiceID]++
		}
	}

	info := map[string]ServiceListInfo{}
	for _, service := range services {
		info[service.ID] = ServiceListInfo{}
		if service.Spec.Mode.Replicated != nil && service.Spec.Mode.Replicated.Replicas != nil {
			info[service.ID] = ServiceListInfo{
				Mode:     "replicated",
				Replicas: fmt.Sprintf("%d/%d", running[service.ID], *service.Spec.Mode.Replicated.Replicas),
			}
		} else if service.Spec.Mode.Global != nil {
			info[service.ID] = ServiceListInfo{
				Mode:     "global",
				Replicas: fmt.Sprintf("%d/%d", running[service.ID], tasksNoShutdown[service.ID]),
			}
		}
	}
	return info
}

// ServiceListInfo stores the information about mode and replicas to be used by template
type ServiceListInfo struct {
	Mode     string
	Replicas string
}
