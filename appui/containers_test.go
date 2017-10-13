package appui

import (
	"sort"
	"testing"

	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/mocks"
	"github.com/moncho/dry/ui"
	drytermui "github.com/moncho/dry/ui/termui"
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
	w.prepareForRendering()
	rows := w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}
	if rows[0].container.ID != "0" || rows[4].container.ID != "4" {
		t.Errorf("First or last container row are not correct. First ID: %s, Last Id: %s", rows[0].container.ID, rows[4].container.ID)
	}

	ui.ActiveScreen.Cursor.ScrollCursorDown()
	w.prepareForRendering()
	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}
	if rows[0].container.ID != "0" || rows[4].container.ID != "4" {
		t.Errorf("First or last container row are not correct. First ID: %s, Last Id: %s", rows[0].container.ID, rows[4].container.ID)
	}

	ui.ActiveScreen.Cursor.ScrollTo(10)
	w.prepareForRendering()
	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}
	if rows[0].container.ID != "5" || rows[4].container.ID != "9" {
		t.Errorf("First or last container row are not correct. First ID: %s, Last Id: %s", rows[0].container.ID, rows[4].container.ID)
	}

	ui.ActiveScreen.Cursor.ScrollCursorUp()
	w.prepareForRendering()
	rows = w.visibleRows()
	if len(rows) != w.height {
		t.Errorf("There is room for %d rows but found %d", w.height, len(rows))
	}
	if rows[0].container.ID != "5" || rows[4].container.ID != "9" {
		t.Errorf("First or last container row are not correct. First ID: %s, Last Id: %s", rows[0].container.ID, rows[4].container.ID)
	}

	ui.ActiveScreen.Cursor.ScrollTo(0)
	w.prepareForRendering()
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

	w.prepareForRendering()
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
			if s.mounted {
				t.Errorf("Widget is still mounted")
			}
		})
	}
}

func TestContainersWidget_sortRows(t *testing.T) {
	type fields struct {
		totalRows []*ContainerRow
		sortMode  docker.SortMode
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			"sort by container ID",
			fields{
				[]*ContainerRow{
					{
						ID: drytermui.NewThemedParColumn(&ui.ColorTheme{}, "2"),
					},
					{
						ID: drytermui.NewThemedParColumn(&ui.ColorTheme{}, "1"),
					},
					{
						ID: drytermui.NewThemedParColumn(&ui.ColorTheme{}, "3"),
					},
				},
				docker.SortByContainerID,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ContainersWidget{
				totalRows: tt.fields.totalRows,
				sortMode:  tt.fields.sortMode,
			}
			s.sortRows()

			if !sort.SliceIsSorted(s.totalRows,
				func(i, j int) bool {
					return s.totalRows[i].ID.Text < s.totalRows[j].ID.Text
				}) {
				t.Error("rows are not sorted")
			}
		})
	}
}

func TestContainersWidget_filterRows(t *testing.T) {
	type fields struct {
		totalRows     []*ContainerRow
		filteredRows  []*ContainerRow
		filterPattern string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			"filter test",
			fields{
				[]*ContainerRow{
					{
						ID:      drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Image:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Names:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Command: drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
					},
					{
						ID:      drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Image:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Names:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Command: drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
					},
					{
						ID:      drytermui.NewThemedParColumn(&ui.ColorTheme{}, "yup"),
						Image:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Names:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Command: drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope")},
					{
						ID:      drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Image:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Names:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Command: drytermui.NewThemedParColumn(&ui.ColorTheme{}, "yup"),
					},
				},
				[]*ContainerRow{
					{
						ID:      drytermui.NewThemedParColumn(&ui.ColorTheme{}, "yup"),
						Image:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Names:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Command: drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope")},
					{
						ID:      drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Image:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Names:   drytermui.NewThemedParColumn(&ui.ColorTheme{}, "nope"),
						Command: drytermui.NewThemedParColumn(&ui.ColorTheme{}, "yup"),
					},
				},
				"yup",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ContainersWidget{
				totalRows:     tt.fields.totalRows,
				filterPattern: tt.fields.filterPattern,
			}
			s.filterRows()
			if len(s.filteredRows) != len(tt.fields.filteredRows) {
				t.Errorf("Filtering not working, expected: %v, got: %v", tt.fields.filteredRows, s.filteredRows)
			}
		})
	}
}
