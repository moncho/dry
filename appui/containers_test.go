package appui

import (
	"testing"

	"github.com/moncho/dry/mocks"
	"github.com/moncho/dry/ui"
)

func TestContainerListVisibleRows(t *testing.T) {

	daemon := &mocks.DockerDaemonMock{}
	screen := &ui.Screen{
		Cursor:     &ui.Cursor{},
		Dimensions: &ui.Dimensions{Height: 16, Width: 40},
	}
	//DockerDaemonMock returns 10 running 1containers
	screen.Cursor.Max(10 - 1)
	ui.ActiveScreen = screen

	w := NewContainersWidget(daemon, 0)

	if err := w.Mount(); err != nil {
		t.Errorf("There was an error mounting the widget %v", err)
	}

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

func TestContainersWidget_ToggleShowAllContainers(t *testing.T) {
	type fields struct {
		showAllContainers bool
		mounted           bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			"ToggleShowAllContainers",
			fields{
				true,
				true,
			},
		},
		{
			"ToggleShowAllContainers always changes the widget state to unmounted",
			fields{
				false,
				true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ContainersWidget{
				showAllContainers: tt.fields.showAllContainers,
				mounted:           tt.fields.mounted,
			}
			s.ToggleShowAllContainers()
			if s.showAllContainers == tt.fields.showAllContainers {
				t.Error("Show all containers state did not change after toggle")
			}
			if s.mounted != false {
				t.Errorf("Widget is still mounted")
			}
		})
	}
}
