package appui

import (
	"sync"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
)

//SwarmNodesWidget presents Docker swarm information
type SwarmNodesWidget struct {
	swarmClient docker.SwarmAPI
	nodes       []*NodeRow
	sync.RWMutex
}

//NewSwarmNodesWidget creates a SwarmNodesWidget
func NewSwarmNodesWidget(swarmClient docker.SwarmAPI) *SwarmNodesWidget {
	w := &SwarmNodesWidget{swarmClient: swarmClient}
	if nodes, err := swarmClient.SwarmNodes(); err == nil {
		for _, node := range nodes {
			w.nodes = append(w.nodes, NewNodeRow(node))
		}
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
