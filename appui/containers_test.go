package appui

import (
	"strconv"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

func TestContainerListVisibleRows(t *testing.T) {
	var containers []*docker.Container
	for index := 0; index < 10; index++ {
		containers = append(containers, &docker.Container{
			Container: types.Container{ID: strconv.Itoa(index), Names: []string{"Name"}, Status: "Never worked"},
		})
	}
	screen := &ui.Screen{
		Cursor:     &ui.Cursor{},
		Dimensions: &ui.Dimensions{Height: 16, Width: 40},
	}
	screen.Cursor.Max(len(containers) - 1)
	ui.ActiveScreen = screen

	w := NewContainersWidget(&DockerPsRenderData{containers, docker.SortByContainerID}, 0)

	rows := w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}

	ui.ActiveScreen.Cursor.ScrollCursorDown()

	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}

	ui.ActiveScreen.Cursor.ScrollTo(10)
	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}

	ui.ActiveScreen.Cursor.ScrollCursorUp()
	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}

	ui.ActiveScreen.Cursor.ScrollTo(0)
	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}

}
