package docker

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/docker/mock"
)

func TestImageRun(t *testing.T) {
	daemon := DockerDaemon{client: mock.ImageAPIClientMock{}}
	err := daemon.RunImage(&types.ImageSummary{}, "command")

	if err != nil {
		t.Errorf("Running an image with the list of swarm nodes resulted in error: %s", err.Error())
	}
}
