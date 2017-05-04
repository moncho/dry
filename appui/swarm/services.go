package swarm

import (
	"fmt"
	"sync"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/cli/command/formatter"
	"github.com/docker/docker/cli/command/service"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"

	gizaktermui "github.com/gizak/termui"
)

var defaultServiceTableHeader = serviceTableHeader()

//ServicesWidget shows information about services running on the Swarm
type ServicesWidget struct {
	swarmClient          docker.SwarmAPI
	services             []*ServiceRow
	header               *termui.TableHeader
	selectedIndex        int
	offset               int
	x, y                 int
	height, width        int
	startIndex, endIndex int
	sync.RWMutex
}

//NewServicesWidget creates a ServicesWidget
func NewServicesWidget(swarmClient docker.SwarmAPI, y int) *ServicesWidget {
	w := &ServicesWidget{
		swarmClient:   swarmClient,
		header:        defaultServiceTableHeader,
		selectedIndex: 0,
		offset:        0,
		x:             0,
		y:             y,
		height:        appui.MainScreenAvailableHeight(),
		width:         ui.ActiveScreen.Dimensions.Width}
	if services, servicesInfo, err := getServiceInfo(swarmClient); err == nil {
		for _, service := range services {
			w.services = append(w.services, NewServiceRow(service, servicesInfo[service.ID], w.header))
		}
	}
	w.align()

	return w

}

//Align aligns rows
func (s *ServicesWidget) align() {
	y := s.y
	x := s.x
	width := s.width

	s.header.SetWidth(width)
	s.header.SetY(y)
	s.header.SetX(x)

	for _, service := range s.services {
		service.SetX(x)
		service.SetWidth(width)
	}

}

//Buffer returns the content of this widget as a termui.Buffer
func (s *ServicesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()

	buf := gizaktermui.NewBuffer()
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
func (s *ServicesWidget) RowCount() int {
	return len(s.services)
}
func (s *ServicesWidget) highlightSelectedRow() {
	if s.RowCount() == 0 {
		return
	}
	index := ui.ActiveScreen.Cursor.Position()
	if index > s.RowCount() {
		index = s.RowCount() - 1
	}
	s.services[s.selectedIndex].NotHighlighted()
	s.selectedIndex = index
	s.services[s.selectedIndex].Highlighted()
}

//OnEvent runs the given command
func (s *ServicesWidget) OnEvent(event appui.EventCommand) error {
	return event(s.services[s.selectedIndex].service.ID)
}

func (s *ServicesWidget) visibleRows() []*ServiceRow {

	//no screen
	if s.height < 0 {
		return nil
	}
	rows := s.services
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

func serviceTableHeader() *termui.TableHeader {
	fields := []string{
		"ID", "NAME", "MODE", "REPLICAS", "SERVICE PORT(S)", "IMAGE"}

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	header.AddColumn(fields[0])
	header.AddColumn(fields[1])
	header.AddFixedWidthColumn(fields[2], 12)
	header.AddFixedWidthColumn(fields[3], 10)
	header.AddColumn(fields[4])
	header.AddColumn(fields[5])

	return header
}

func getServiceInfo(swarmClient docker.SwarmAPI) ([]swarm.Service, map[string]formatter.ServiceListInfo, error) {

	serviceFilters := filters.NewArgs()
	serviceFilters.Add("runtime", string(swarm.RuntimeContainer))
	services, err := swarmClient.Services()
	if err != nil {
		return nil, nil, err
	}

	info := map[string]formatter.ServiceListInfo{}
	if len(services) > 0 {

		tasks, err := swarmClient.ServiceTasks(serviceIDs(services)...)
		if err != nil {
			return nil, nil, err
		}

		nodes, err := swarmClient.Nodes()
		if err != nil {
			return nil, nil, err
		}

		info = service.GetServicesStatus(services, nodes, tasks)
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
func getServicesStatus(services []swarm.Service, nodes []swarm.Node, tasks []swarm.Task) map[string]formatter.ServiceListInfo {
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

	info := map[string]formatter.ServiceListInfo{}
	for _, service := range services {
		info[service.ID] = formatter.ServiceListInfo{}
		if service.Spec.Mode.Replicated != nil && service.Spec.Mode.Replicated.Replicas != nil {
			info[service.ID] = formatter.ServiceListInfo{
				Mode:     "replicated",
				Replicas: fmt.Sprintf("%d/%d", running[service.ID], *service.Spec.Mode.Replicated.Replicas),
			}
		} else if service.Spec.Mode.Global != nil {
			info[service.ID] = formatter.ServiceListInfo{
				Mode:     "global",
				Replicas: fmt.Sprintf("%d/%d", running[service.ID], tasksNoShutdown[service.ID]),
			}
		}
	}
	return info
}
