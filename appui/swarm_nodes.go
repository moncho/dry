package appui

import (
	"sync"

	"github.com/docker/docker/api/types/swarm"
	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
)

//SwarmNodesWidget presents Docker swarm information
type SwarmNodesWidget struct {
	swarmClient docker.SwarmAPI
	nodes       []swarm.Node
	sync.RWMutex
}

//NewSwarmNodesWidget creates a SwarmNodesWidget
func NewSwarmNodesWidget(swarmClient docker.SwarmAPI) *SwarmNodesWidget {
	w := &SwarmNodesWidget{swarmClient: swarmClient}
	if nodes, err := swarmClient.SwarmNodes(); err == nil {
		w.nodes = nodes
	}

	return w
}

//Buffer returns the content of this widget as a termui.Buffer
func (s *SwarmNodesWidget) Buffer() gizaktermui.Buffer {
	s.Lock()
	defer s.Unlock()

	buf := gizaktermui.NewBuffer()

	return buf
}
