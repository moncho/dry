package docker

import (
	"testing"

	"github.com/docker/engine-api/types"
)

func TestImageNameFormatting(t *testing.T) {
	formatter := ImageFormatter{
		trunc: false,
		image: types.Image{
			RepoTags: []string{"nginx:1.10.0-alpine"},
		},
	}

	repo := formatter.Repository()
	tag := formatter.Tag()
	if repo != "nginx" {
		t.Errorf("Repo value not what expected after formatting: %s", repo)
	} else if tag != "1.10.0-alpine" {
		t.Errorf("Tag value not what expected after formatting: %s", tag)
	}
}
func TestImageNameFormattingPrivateRegistry(t *testing.T) {
	formatter := ImageFormatter{
		trunc: false,
		image: types.Image{
			RepoTags: []string{"localhost:5000/nginx:1.10.0-alpine"},
		},
	}

	repo := formatter.Repository()
	tag := formatter.Tag()
	if repo != "localhost:5000/nginx" {
		t.Errorf("Repo value not what expected after formatting: %s", repo)
	} else if tag != "1.10.0-alpine" {
		t.Errorf("Tag value not what expected after formatting: %s", tag)
	}
}
