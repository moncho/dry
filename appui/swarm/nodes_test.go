package swarm

import (
	"image"
	"testing"

	"github.com/moncho/dry/mocks"
	"github.com/moncho/dry/ui"
)

type testScreen struct {
	cursor     *ui.Cursor
	dimensions ui.Dimensions
}

func (ts *testScreen) Cursor() *ui.Cursor {
	return ts.cursor
}
func (ts *testScreen) Bounds() image.Rectangle {
	y := 1
	return image.Rect(0, y, ts.dimensions.Width, y+ts.dimensions.Height)
}

func TestNodesWidgetCreation(t *testing.T) {
	screen := &testScreen{
		dimensions: ui.Dimensions{Height: 14, Width: 100},
		cursor:     ui.NewCursor()}
	w := NewNodesWidget(&mocks.SwarmDockerDaemon{}, screen)
	if w == nil {
		t.Error("Swarm widget is nil")
	}
	if w.swarmClient == nil {
		t.Error("Swarm widget does not have a reference to the swarmclient")
	}

	if w.header.ColumnCount() != len(nodeTableHeaders) {
		t.Error("Swarm widget does not have a the expected number of columns")
	}

}

func TestNodesWidgetMount(t *testing.T) {
	screen := &testScreen{
		dimensions: ui.Dimensions{Height: 14, Width: 100},
		cursor:     ui.NewCursor()}
	w := NewNodesWidget(&mocks.SwarmDockerDaemon{}, screen)

	if len(w.totalRows) != 0 {
		t.Errorf("Swarm widget is not showing the expected number of totalRows. Got: %d", len(w.totalRows))
	}

	w.Mount()

	if len(w.totalRows) != 1 {
		t.Errorf("Swarm widget is not showing the expected number of totalRows. Got: %d", len(w.totalRows))
	}

}
