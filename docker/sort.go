package docker

import "sort"

// Allowed sort methods
const (
	NoSort SortMode = iota
	SortByContainerID
	SortByImage
	SortByStatus
	SortByName
)

// SortMode represents allowed modes to sort a container slice
type SortMode uint16

type apiContainers []*Container

func (a apiContainers) Len() int      { return len(a) }
func (a apiContainers) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type byContainerID struct{ apiContainers }

func (a byContainerID) Less(i, j int) bool { return a.apiContainers[i].ID < a.apiContainers[j].ID }

type byImage struct{ apiContainers }

func (a byImage) Less(i, j int) bool {
	//If the image is the same, sorting is done by name
	if a.apiContainers[i].Image == a.apiContainers[j].Image {
		return byName(a).Less(i, j)
	}
	return a.apiContainers[i].Image < a.apiContainers[j].Image
}

type byStatus struct{ apiContainers }

func (a byStatus) Less(i, j int) bool {
	//If the status is the same, sorting is done by name
	if a.apiContainers[i].Status == a.apiContainers[j].Status {
		return byName(a).Less(i, j)
	}
	return a.apiContainers[i].Status < a.apiContainers[j].Status
}

type byName struct{ apiContainers }

func (a byName) Less(i, j int) bool {
	if len(a.apiContainers[i].Names) > 0 {
		if len(a.apiContainers[j].Names) > 0 {
			return a.apiContainers[i].Names[0] < a.apiContainers[j].Names[0]
		}
		return true
	}
	return false
}

// SortContainers sorts the given containers slice using the given mode
func SortContainers(containers []*Container, mode SortMode) {
	switch mode {
	case NoSort:
	case SortByContainerID:
		sort.Sort(byContainerID{containers})
	case SortByImage:
		sort.Sort(byImage{containers})
	case SortByStatus:
		sort.Sort(byStatus{containers})
	case SortByName:
		sort.Sort(byName{containers})
	}
}
