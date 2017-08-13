package docker

import (
	"errors"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"golang.org/x/net/context"
)

//History returns image history
func (daemon *DockerDaemon) History(id string) ([]image.HistoryResponseItem, error) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	return daemon.client.ImageHistory(
		ctx, id)
}

//ImageAt returns the Image found at the given
//position.
func (daemon *DockerDaemon) ImageAt(pos int) (*dockerTypes.ImageSummary, error) {
	daemon.imagesLock.Lock()
	defer daemon.imagesLock.Unlock()
	if pos >= len(daemon.images) {
		return nil, errors.New("Position is higher than number of images")
	}
	return &daemon.images[pos], nil
}

//Images returns the list of Docker images
func (daemon *DockerDaemon) Images() ([]dockerTypes.ImageSummary, error) {
	daemon.imagesLock.Lock()
	defer daemon.imagesLock.Unlock()
	return daemon.images, nil
}

//ImagesCount returns the number of images
func (daemon *DockerDaemon) ImagesCount() int {
	daemon.imagesLock.Lock()
	defer daemon.imagesLock.Unlock()
	return len(daemon.images)
}

func (daemon *DockerDaemon) RunImage(image *dockerTypes.ImageSummary, command string) error {
	return nil
}
