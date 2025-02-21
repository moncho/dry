package docker

import (
	"testing"

	"github.com/docker/docker/api/types/image"
	"github.com/moncho/dry/docker/mock"
)

func TestImageRun(t *testing.T) {
	daemon := DockerDaemon{client: mock.ImageAPIClientMock{}}
	err := daemon.RunImage(image.Summary{
		RepoTags: []string{"nope:latest"},
	}, "command")

	if err != nil {
		t.Errorf("Running an image resulted in error %s", err.Error())
	}
}
