package appui

import (
	"sync"

	drytermui "github.com/moncho/dry/ui/termui"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//ContainerMenuWidget shows the actions menu of a container
type ContainerMenuWidget struct {
	dockerDaemon  docker.ContainerAPI
	rows          []*Row
	cID           string
	height, width int
	mounted       bool
	selectedIndex int
	x, y          int
	sync.RWMutex
}

//NewContainerMenuWidget creates a TasksWidget
func NewContainerMenuWidget(dockerDaemon docker.ContainerAPI, y int) *ContainerMenuWidget {
	w := ContainerMenuWidget{
		height: MainScreenAvailableHeight(),
		width:  30,
		y:      y,
	}
	return &w
}

//Buffer returns the content of this widget as a termui.Buffer
func (s *ContainerMenuWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()
	buf := gizaktermui.NewBuffer()
	if s.mounted {
		y := s.y
		s.prepareForRendering()

		for i, row := range s.rows {
			row.SetY(y)
			y += row.GetHeight()
			if i != s.selectedIndex {
				row.NotHighlighted()
			} else {
				row.Highlighted()
			}
			buf.Merge(row.Buffer())
		}

	}
	return buf
}

//Filter is a noop for this widget
func (s *ContainerMenuWidget) Filter(filter string) {

}

//ForContainer sets the container for which this widget is showing tasks
func (s *ContainerMenuWidget) ForContainer(cID string) {
	s.Lock()
	defer s.Unlock()

	s.cID = cID
	s.mounted = false

}

//Mount prepares this widget for rendering
func (s *ContainerMenuWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if !s.mounted {
		rows := make([]*Row, len(docker.CommandDescriptions))
		for i, command := range docker.CommandDescriptions {
			r := &Row{
				ParColumns: []*drytermui.ParColumn{drytermui.NewThemedParColumn(DryTheme, command)},
			}
			r.AddColumn(r.ParColumns[0])
			r.Height = 1
			r.Width = r.ParColumns[0].Width
			rows[i] = r
		}
		s.rows = rows
		s.align()
		s.mounted = true
	}
	return nil
}

//Name returns this widget name
func (s *ContainerMenuWidget) Name() string {
	return "ContainerMenuWidget"
}

//OnEvent runs the given command
func (s *ContainerMenuWidget) OnEvent(event EventCommand) error {
	return event(s.cID + ":" + s.rows[s.selectedIndex].ParColumns[0].Text)
}

//RowCount returns the number of rows of this widget.
func (s *ContainerMenuWidget) RowCount() int {
	return len(s.rows)
}

//Sort is a noop for this widget
func (s *ContainerMenuWidget) Sort() {
}

//Unmount this widget
func (s *ContainerMenuWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()
	s.mounted = false
	return nil
}

func (s *ContainerMenuWidget) align() {
	x := s.x
	width := s.width

	for _, row := range s.rows {
		row.SetX(x)
		row.SetWidth(width)
	}
}

func (s *ContainerMenuWidget) prepareForRendering() {
	index := ui.ActiveScreen.Cursor.Position()
	if index < 0 {
		index = 0
	} else if index > len(s.rows) {
		index = len(s.rows) - 1
	}
	s.selectedIndex = index
}
