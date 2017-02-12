package docker

import (
	"sort"

	"github.com/docker/docker/api/types"
)

//Allowed sort methods
const (
	NoSortNetworks SortNetworksMode = iota
	SortNetworksByID
	SortNetworksByName
	SortNetworksByDriver
)

//SortNetworksMode represents allowed modes to sort Docker images
type SortNetworksMode uint16

type dockerNetworks []types.NetworkResource

func (s dockerNetworks) Len() int      { return len(s) }
func (s dockerNetworks) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type networksByID struct{ dockerNetworks }

func (s networksByID) Less(i, j int) bool { return s.dockerNetworks[i].ID < s.dockerNetworks[j].ID }

type networksByName struct{ dockerNetworks }

func (s networksByName) Less(i, j int) bool {
	if len(s.dockerNetworks[i].Name) > 0 {
		if len(s.dockerNetworks[j].Name) > 0 {
			return s.dockerNetworks[i].Name[0] < s.dockerNetworks[j].Name[0]
		}
		return true
	}
	return false
}

type networksByDriver struct{ dockerNetworks }

func (s networksByDriver) Less(i, j int) bool {
	return s.dockerNetworks[i].Driver < s.dockerNetworks[j].Driver
}

//SortNetworks sorts the given network slice using the given mode
func SortNetworks(images []types.NetworkResource, mode SortNetworksMode) {
	switch mode {
	case SortNetworksByID:
		sort.Sort(networksByID{images})
	case SortNetworksByName:
		sort.Sort(networksByName{images})
	case SortNetworksByDriver:
		sort.Sort(networksByDriver{images})
	}
}
