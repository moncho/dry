package appui

import (
	"testing"

	"github.com/moncho/dry/mocks"
)

func TestSwarmNodesWidget(t *testing.T) {

	w := NewSwarmNodesWidget(&mocks.SwarmDockerDaemon{}, 1)
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
