package docker

import (
	"sort"
)

// Allowed sort methods
const (
	NoSortStack SortMode = iota
	SortByStackName
)

type swarmStacks []Stack

func (s swarmStacks) Len() int      { return len(s) }
func (s swarmStacks) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type stacksByName struct{ swarmStacks }

func (s stacksByName) Less(i, j int) bool {

	return s.swarmStacks[i].Name < s.swarmStacks[j].Name
}

// SortStacks sorts the given stack slice using the given mode
func SortStacks(stacks []Stack, mode SortMode) {
	switch mode {
	case SortByStackName:
		sort.SliceStable(stacksByName{stacks}.swarmStacks, stacksByName{stacks}.Less)
	}

}
