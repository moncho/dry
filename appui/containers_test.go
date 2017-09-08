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

	w := NewContainersWidget(0)

	w.PrepareToRender(&DockerPsRenderData{containers, docker.SortByContainerID, ""})

	rows := w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}
	if rows[0].container.ID != "0" || rows[4].container.ID != "4" {
		t.Errorf("First or last container row are not correct. First ID: %s, Last Id: %s", rows[0].container.ID, rows[4].container.ID)
	}

	ui.ActiveScreen.Cursor.ScrollCursorDown()
	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}
	if rows[0].container.ID != "0" || rows[4].container.ID != "4" {
		t.Errorf("First or last container row are not correct. First ID: %s, Last Id: %s", rows[0].container.ID, rows[4].container.ID)
	}

	ui.ActiveScreen.Cursor.ScrollTo(10)
	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}
	if rows[0].container.ID != "5" || rows[4].container.ID != "9" {
		t.Errorf("First or last container row are not correct. First ID: %s, Last Id: %s", rows[0].container.ID, rows[4].container.ID)
	}

	ui.ActiveScreen.Cursor.ScrollCursorUp()
	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}
	if rows[0].container.ID != "5" || rows[4].container.ID != "9" {
		t.Errorf("First or last container row are not correct. First ID: %s, Last Id: %s", rows[0].container.ID, rows[4].container.ID)
	}

	ui.ActiveScreen.Cursor.ScrollTo(0)
	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}
	if rows[0].container.ID != "0" || rows[4].container.ID != "4" {
		t.Errorf("First or last container row are not correct. First ID: %s, Last Id: %s", rows[0].container.ID, rows[4].container.ID)
	}

	ui.ActiveScreen.Cursor.ScrollCursorDown()
	ui.ActiveScreen.Cursor.ScrollCursorDown()
	ui.ActiveScreen.Cursor.ScrollCursorDown()
	ui.ActiveScreen.Cursor.ScrollCursorDown()
	ui.ActiveScreen.Cursor.ScrollCursorDown()

	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}
	if rows[0].container.ID != "1" || rows[4].container.ID != "5" {
		t.Errorf("First or last container row are not correct. First ID: %s, Last Id: %s", rows[0].container.ID, rows[4].container.ID)
	}

}
