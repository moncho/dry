package swarm

import (
	"testing"

	"github.com/moncho/dry/mocks"
	"github.com/moncho/dry/ui"
)

func TestNodesWidgetCreation(t *testing.T) {
	ui.ActiveScreen = &ui.Screen{
		Dimensions: &ui.Dimensions{Height: 14, Width: 100},
		Cursor:     ui.NewCursor()}
	w := NewNodesWidget(&mocks.SwarmDockerDaemon{}, 1)
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
	ui.ActiveScreen = &ui.Screen{
		Dimensions: &ui.Dimensions{Height: 14, Width: 100},
		Cursor:     ui.NewCursor()}
	w := NewNodesWidget(&mocks.SwarmDockerDaemon{}, 1)

	if len(w.totalRows) != 0 {
		t.Errorf("Swarm widget is not showing the expected number of totalRows. Got: %d", len(w.totalRows))
	}

	w.Mount()

	if len(w.totalRows) != 1 {
		t.Errorf("Swarm widget is not showing the expected number of totalRows. Got: %d", len(w.totalRows))
	}

}
