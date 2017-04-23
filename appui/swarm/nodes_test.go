package swarm

import (
	"testing"

	"github.com/moncho/dry/mocks"
	"github.com/moncho/dry/ui"
)

func TestNodesWidget(t *testing.T) {
	ui.ActiveScreen = &ui.Screen{
		Dimensions: &ui.Dimensions{Height: 14, Width: 100},
		Cursor:     ui.NewCursor()}
	w := NewNodesWidget(&mocks.SwarmDockerDaemon{}, 1)
	if w == nil {
		t.Error("Swarm widget is nil")
	}
	if len(w.nodes) != 1 {
		t.Errorf("Swarm widget is not showing the expected number of nodes. Got: %d", len(w.nodes))
	}
	if w.swarmClient == nil {
		t.Error("Swarm widget does not have a reference to the swarmclient")
	}

}
