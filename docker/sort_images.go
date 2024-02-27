package docker

import (
	"sort"

	"github.com/docker/docker/api/types"
)

// Allowed sort methods
const (
	NoSortImages SortMode = iota
	SortImagesByID
	SortImagesByRepo
	SortImagesBySize
	SortImagesByCreationDate
)

type apiImages []types.ImageSummary

func (s apiImages) Len() int      { return len(s) }
func (s apiImages) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type byID struct{ apiImages }

func (s byID) Less(i, j int) bool { return s.apiImages[i].ID < s.apiImages[j].ID }

type byRepository struct{ apiImages }

func (s byRepository) Less(i, j int) bool {
	if len(s.apiImages[i].RepoTags) > 0 {
		if len(s.apiImages[j].RepoTags) > 0 {
			return s.apiImages[i].RepoTags[0] < s.apiImages[j].RepoTags[0]
		}
		return true
	}
	return false
}

type bySize struct{ apiImages }

func (s bySize) Less(i, j int) bool {
	return s.apiImages[i].Size < s.apiImages[j].Size
}

type byCreationDate struct{ apiImages }

func (s byCreationDate) Less(i, j int) bool {
	//More recent first
	return s.apiImages[i].Created > s.apiImages[j].Created
}

// SortImages sorts the given image slice using the given mode
func SortImages(images []types.ImageSummary, mode SortMode) {
	switch mode {
	case SortImagesByID:
		sort.Sort(byID{images})
	case SortImagesByRepo:
		sort.Sort(byRepository{images})
	case SortImagesBySize:
		sort.Sort(bySize{images})
	case SortImagesByCreationDate:
		sort.Sort(byCreationDate{images})
	}
}
