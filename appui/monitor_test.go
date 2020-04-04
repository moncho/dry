package appui

import (
	"image"
	"testing"

	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

type dockerMonitor struct {
}

func (dockerMonitor) Containers(filters []docker.ContainerFilter, mode docker.SortMode) []*docker.Container {
	return nil
}

func (dockerMonitor) StatsChannel(container *docker.Container) (*docker.StatsChannel, error) {
	return nil, nil
}

type screenBuffererRender struct {
}

func (screenBuffererRender) Bounds() image.Rectangle {
	return image.Rectangle{}
}
func (screenBuffererRender) Cursor() *ui.Cursor {
	return &ui.Cursor{}
}

func (screenBuffererRender) Flush() *ui.Screen {
	return nil
}
func (screenBuffererRender) RenderBufferer(bs ...termui.Bufferer) {

}

func TestMonitor_RepeatedUnmount(t *testing.T) {
	type fields struct {
		daemon   DockerMonitor
		renderer ScreenBuffererRender
	}
	m := NewMonitor(dockerMonitor{}, screenBuffererRender{})
	m.Mount()
	m.Unmount()
	m.Unmount()
}
