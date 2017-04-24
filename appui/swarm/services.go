package swarm

import (
	"sync"

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
	if services, err := swarmClient.Services(); err == nil {
		for _, service := range services {
			w.services = append(w.services, NewServiceRow(service))
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
	s.selectedIndex = index
}

//OnEvent runs the given command
func (s *ServicesWidget) OnEvent(event appui.EventCommand) error {
	return nil
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
		"ID", "NAME", "MODE", "REPLICAS", "IMAGE"}

	header := termui.NewHeader(appui.DryTheme)
	header.ColumnSpacing = appui.DefaultColumnSpacing
	for _, f := range fields {
		header.AddColumn(f)
	}
	return header
}
