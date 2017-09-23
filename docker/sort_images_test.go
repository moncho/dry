package docker

import (
	"testing"

	"github.com/docker/docker/api/types"
)

func TestSortImages(t *testing.T) {
	type args struct {
		images []types.ImageSummary
		mode   SortMode
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Sort by repo ",
			args{
				[]types.ImageSummary{
					types.ImageSummary{
						ID:       "1",
						RepoTags: []string{"dry/dry:2"},
					},
					types.ImageSummary{
						ID:       "0",
						RepoTags: []string{"dry/dry:1"},
					},
				},
				SortImagesByRepo,
			},
		},
		{
			"Sort by ID ",
			args{
				[]types.ImageSummary{
					types.ImageSummary{
						ID: "3",
					},
					types.ImageSummary{
						ID: "1",
					},
					types.ImageSummary{
						ID: "0",
					},
				},
				SortImagesByID,
			},
		},
		{
			"Sort by Creation Date ",
			args{
				[]types.ImageSummary{
					types.ImageSummary{
						ID:      "1",
						Created: 1,
					},
					types.ImageSummary{
						ID:      "3",
						Created: 2,
					},
					types.ImageSummary{
						ID:      "0",
						Created: 3,
					},
				},
				SortImagesByCreationDate,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images := tt.args.images
			SortImages(images, tt.args.mode)
			if images[0].ID != "0" {
				t.Errorf("Unexpected image as first element %v", images[0])
			}
			if images[0].ID > images[1].ID {
				t.Errorf("Unexpected order %v", images)
			}
		})
	}
}
