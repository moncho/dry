package docker

import (
	"sort"

	"github.com/fsouza/go-dockerclient"
)

//Allowed sort methods
const (
	NoSort SortMode = iota
	SortByContainerID
	SortByImage
	SortByStatus
	SortByName
)

//SortMode represents allowed modes to sort a container slice
type SortMode uint16

//TODO figure out how to avoid so much duplicated code
type byContainerID []docker.APIContainers
type byImage []docker.APIContainers
type byStatus []docker.APIContainers
type byName []docker.APIContainers

func (a byContainerID) Len() int           { return len(a) }
func (a byContainerID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byContainerID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (a byImage) Len() int      { return len(a) }
func (a byImage) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byImage) Less(i, j int) bool {
	//If the image is the same, sorting is done by name
	if a[i].Image == a[j].Image {
		if len(a[i].Names) > 0 {
			if len(a[j].Names) > 0 {
				return a[i].Names[0] < a[j].Names[0]
			}
		}
	}
	return a[i].Image < a[j].Image
}

func (a byStatus) Len() int           { return len(a) }
func (a byStatus) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byStatus) Less(i, j int) bool { return a[i].Status < a[j].Status }

func (a byName) Len() int      { return len(a) }
func (a byName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool {
	if len(a[i].Names) > 0 {
		if len(a[j].Names) > 0 {
			return a[i].Names[0] < a[j].Names[0]
		}
		return true
	}
	return false
}

//SortContainers sorts the given containers slice using the given mode
func SortContainers(containers []docker.APIContainers, mode SortMode) {
	switch mode {
	case SortByContainerID:
		sort.Sort(byContainerID(containers))
	case SortByImage:
		sort.Sort(byImage(containers))
	case SortByStatus:
		sort.Sort(byStatus(containers))
	case SortByName:
		sort.Sort(byName(containers))
	}
}
