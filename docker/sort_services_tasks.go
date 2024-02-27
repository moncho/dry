package docker

import (
	"sort"

	"github.com/docker/docker/api/types/swarm"
)

// Allowed sort methods
const (
	NoSortTask SortMode = iota
	SortByTaskImage
	SortByTaskService
	SortByTaskState
	SortByTaskDesiredState
)

type swarmTasks []swarm.Task

func (s swarmTasks) Len() int      { return len(s) }
func (s swarmTasks) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type tasksByImage struct{ swarmTasks }

func (s tasksByImage) Less(i, j int) bool {
	return s.swarmTasks[i].Spec.ContainerSpec.Image < s.swarmTasks[j].Spec.ContainerSpec.Image
}

type tasksByService struct{ swarmTasks }

func (s tasksByService) Less(i, j int) bool {
	return s.swarmTasks[i].ServiceID < s.swarmTasks[j].ServiceID
}

type tasksByState struct{ swarmTasks }

func (s tasksByState) Less(i, j int) bool {
	return s.swarmTasks[i].Status.State < s.swarmTasks[j].Status.State
}

type tasksByDesiredState struct{ swarmTasks }

func (s tasksByDesiredState) Less(i, j int) bool {
	return s.swarmTasks[i].DesiredState < s.swarmTasks[j].DesiredState
}

// SortTasks sorts the given Task slice using the given mode
func SortTasks(tasks []swarm.Task, mode SortMode) {

	switch mode {
	case SortByTaskImage:
		sortingAlg := tasksByImage{tasks}
		sort.SliceStable(sortingAlg.swarmTasks, sortingAlg.Less)
	case SortByTaskService:
		sortingAlg := tasksByService{tasks}
		sort.SliceStable(sortingAlg.swarmTasks, sortingAlg.Less)
	case SortByTaskState:
		sortingAlg := tasksByState{tasks}
		sort.SliceStable(sortingAlg.swarmTasks, sortingAlg.Less)
	case SortByTaskDesiredState:
		sortingAlg := tasksByDesiredState{tasks}
		sort.SliceStable(sortingAlg.swarmTasks, sortingAlg.Less)
	}

}
