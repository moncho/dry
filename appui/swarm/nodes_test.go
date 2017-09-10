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

	if w.header.ColumnCount() != len(nodeTableFields) {
		t.Error("Swarm widget does not have a the expected number of columns")
	}

}

func TestNodesWidgetMount(t *testing.T) {
	ui.ActiveScreen = &ui.Screen{
		Dimensions: &ui.Dimensions{Height: 14, Width: 100},
		Cursor:     ui.NewCursor()}
	w := NewNodesWidget(&mocks.SwarmDockerDaemon{}, 1)

	if len(w.nodes) != 0 {
		t.Errorf("Swarm widget is not showing the expected number of nodes. Got: %d", len(w.nodes))
	}

	w.Mount()

	if len(w.nodes) != 1 {
		t.Errorf("Swarm widget is not showing the expected number of nodes. Got: %d", len(w.nodes))
	}

}
