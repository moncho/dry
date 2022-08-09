package appui

import (
	"sync"

	drytermui "github.com/moncho/dry/ui/termui"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
)

type ContainerAPI interface {
	ContainerByID(string) *docker.Container
}

const cMenuWidth = 30

// ContainerMenuWidget shows the actions menu of a container
type ContainerMenuWidget struct {
	containerAPI  ContainerAPI
	rows          []*Row
	cInfo         *ContainerDetailsWidget
	cID           string
	selectedIndex int
	screen        Screen
	OnUnmount     func() error

	sync.RWMutex
	mounted bool
}

// NewContainerMenuWidget creates a TasksWidget
func NewContainerMenuWidget(containerAPI ContainerAPI, s Screen) *ContainerMenuWidget {
	w := ContainerMenuWidget{
		containerAPI: containerAPI,
		screen:       s,
	}

	return &w
}

// Buffer returns the content of this widget as a termui.Buffer
func (s *ContainerMenuWidget) Buffer() gizaktermui.Buffer {
	s.RLock()
	defer s.RUnlock()

	buf := gizaktermui.NewBuffer()
	if !s.mounted {
		return buf
	}

	y := s.screen.Bounds().Min.Y
	s.prepareForRendering()
	if s.cInfo != nil {
		buf.Merge(s.cInfo.Buffer())
		y += s.cInfo.GetHeight()
	}
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

	return buf
}

// Filter is a noop for this widget
func (s *ContainerMenuWidget) Filter(_ string) {

}

// ForContainer sets the container for which this widget is showing tasks
func (s *ContainerMenuWidget) ForContainer(cID string) {
	s.Lock()
	defer s.Unlock()
	s.cID = cID
	s.mounted = false
}

// Mount prepares this widget for rendering
func (s *ContainerMenuWidget) Mount() error {
	s.Lock()
	defer s.Unlock()
	if s.mounted {
		return nil
	}

	c := s.containerAPI.ContainerByID(s.cID)
	if c != nil {
		s.cInfo = NewContainerDetailsWidget(c, s.screen.Bounds().Min.Y)
	}
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

	return nil
}

// Name returns this widget name
func (s *ContainerMenuWidget) Name() string {
	return "ContainerMenuWidget"
}

// OnEvent runs the given command
func (s *ContainerMenuWidget) OnEvent(event EventCommand) error {
	if s.RowCount() <= 0 || s.selectedIndex < 0 || s.selectedIndex >= s.RowCount() {
		return invalidRow{selected: int(s.selectedIndex), max: s.RowCount()}
	}
	return event(s.cID + ":" + s.rows[s.selectedIndex].ParColumns[0].Text)
}

// RowCount returns the number of rows of this widget.
func (s *ContainerMenuWidget) RowCount() int {
	return len(s.rows)
}

// Sort is a noop for this widget
func (s *ContainerMenuWidget) Sort() {
}

// Unmount this widget
func (s *ContainerMenuWidget) Unmount() error {
	s.Lock()
	defer s.Unlock()
	s.mounted = false
	if s.OnUnmount != nil {
		return s.OnUnmount()
	}
	return nil
}

func (s *ContainerMenuWidget) align() {
	width := s.screen.Bounds().Dx()
	if s.cInfo != nil {
		s.cInfo.SetWidth(width)
		s.cInfo.SetX(s.screen.Bounds().Min.X)
	}
	rowsX := (width - cMenuWidth) / 2
	for _, row := range s.rows {
		row.SetX(rowsX)
		row.SetWidth(cMenuWidth)
	}
}

func (s *ContainerMenuWidget) prepareForRendering() {
	s.screen.Cursor().Max(s.RowCount() - 1)

	index := s.screen.Cursor().Position()
	if index < 0 {
		index = 0
	} else if index > len(s.rows) {
		index = len(s.rows) - 1
	}
	s.selectedIndex = index
}
