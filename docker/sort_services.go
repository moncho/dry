package docker

import (
	"sort"

	"github.com/docker/docker/api/types/swarm"
)

// Allowed sort methods
const (
	NoSortService SortMode = iota
	SortByServiceName
	SortByServiceImage
)

type swarmServices []swarm.Service

func (s swarmServices) Len() int      { return len(s) }
func (s swarmServices) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type servicesByName struct{ swarmServices }

func (s servicesByName) Less(i, j int) bool {

	return s.swarmServices[i].Spec.Name < s.swarmServices[j].Spec.Name
}

type servicesByImage struct{ swarmServices }

func (s servicesByImage) Less(i, j int) bool {
	return s.swarmServices[i].Spec.TaskTemplate.ContainerSpec.Image < s.swarmServices[j].Spec.TaskTemplate.ContainerSpec.Image
}

// SortServices sorts the given service slice using the given mode
func SortServices(services []swarm.Service, mode SortMode) {
	switch mode {
	case SortByServiceName:
		sort.SliceStable(servicesByName{services}.swarmServices, servicesByName{services}.Less)
	case SortByServiceImage:
		sort.SliceStable(servicesByImage{services}.swarmServices, servicesByImage{services}.Less)

	}

}
