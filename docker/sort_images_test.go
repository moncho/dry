package docker

import (
	"testing"

	"github.com/docker/docker/api/types/image"
)

func TestSortImages(t *testing.T) {
	type args struct {
		images []image.Summary
		mode   SortMode
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Sort by repo ",
			args{
				[]image.Summary{
					{
						ID:       "1",
						RepoTags: []string{"dry/dry:2"},
					},
					{
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
				[]image.Summary{
					{
						ID: "3",
					},
					{
						ID: "1",
					},
					{
						ID: "0",
					},
				},
				SortImagesByID,
			},
		},
		{
			"Sort by Creation Date ",
			args{
				[]image.Summary{
					{
						ID:      "1",
						Created: 1,
					},
					{
						ID:      "3",
						Created: 2,
					},
					{
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
