package docker

import (
	"sort"

	"github.com/docker/docker/api/types/swarm"
)

// Allowed sort methods
const (
	NoSortNode SortMode = iota
	SortByNodeName
	SortByNodeRole
	SortByNodeCPU
	SortByNodeMem
	SortByNodeStatus
)

type swarmNodes []swarm.Node

func (s swarmNodes) Len() int      { return len(s) }
func (s swarmNodes) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type nodesByName struct{ swarmNodes }

func (s nodesByName) Less(i, j int) bool {
	return s.swarmNodes[i].Description.Hostname < s.swarmNodes[j].Description.Hostname
}

type nodesByRole struct{ swarmNodes }

func (s nodesByRole) Less(i, j int) bool {
	return s.swarmNodes[i].Spec.Role < s.swarmNodes[j].Spec.Role
}

type nodesByCPU struct{ swarmNodes }

func (s nodesByCPU) Less(i, j int) bool {
	return s.swarmNodes[i].Description.Resources.NanoCPUs < s.swarmNodes[j].Description.Resources.NanoCPUs
}

type nodesByMem struct{ swarmNodes }

func (s nodesByMem) Less(i, j int) bool {
	return s.swarmNodes[i].Description.Resources.MemoryBytes < s.swarmNodes[j].Description.Resources.MemoryBytes
}

type nodesByStatus struct{ swarmNodes }

func (s nodesByStatus) Less(i, j int) bool {
	return s.swarmNodes[i].Status.State < s.swarmNodes[j].Status.State
}

// SortNodes sorts the given nodes slice using the given mode
func SortNodes(nodes []swarm.Node, mode SortMode) {
	switch mode {
	case SortByNodeName:
		sort.Sort(nodesByName{nodes})
	case SortByNodeRole:
		sort.Sort(nodesByRole{nodes})
	case SortByNodeCPU:
		sort.Sort(nodesByCPU{nodes})
	case SortByNodeMem:
		sort.Sort(nodesByMem{nodes})
	case SortByNodeStatus:
		sort.Sort(nodesByStatus{nodes})
	}

}
