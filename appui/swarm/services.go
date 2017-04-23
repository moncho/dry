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

}

//Buffer returns the content of this widget as a termui.Buffer
func (s *ServicesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()

	buf := gizaktermui.NewBuffer()
	buf.Merge(s.header.Buffer())

	y := s.y
	y += s.header.GetHeight()

	return buf
}

//RowCount returns the number of rowns of this widget.
func (s *ServicesWidget) RowCount() int {
	return 0
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
