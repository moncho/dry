package docker

import (
	"sort"

	"github.com/docker/engine-api/types"
)

//Allowed sort methods
const (
	NoSortImages SortImagesMode = iota
	SortImagesByID
	SortImagesByRepo
	SortImagesBySize
	SortImagesByCreationDate
)

//SortImagesMode represents allowed modes to sort Docker images
type SortImagesMode uint16

type apiImages []types.Image

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
	return s.apiImages[i].Created < s.apiImages[j].Created
}

//SortImages sorts the given image slice using the given mode
func SortImages(images []types.Image, mode SortImagesMode) {
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
