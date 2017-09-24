package docker

import (
	"fmt"
	"strings"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/strslice"
	pkgError "github.com/pkg/errors"
	"golang.org/x/net/context"
)

//History returns image history
func (daemon *DockerDaemon) History(id string) ([]image.HistoryResponseItem, error) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	return daemon.client.ImageHistory(
		ctx, id)
}

//ImageByID returns the image with the given ID
func (daemon *DockerDaemon) ImageByID(id string) (dockerTypes.ImageSummary, error) {
	var result dockerTypes.ImageSummary
	images, err := daemon.Images()
	if err != nil {
		return result, err
	}
	for _, image := range images {
		if image.ID == id {
			return image, nil
		}
	}

	return result, fmt.Errorf("Image %s not found", id)

}

//Images returns the list of Docker images
func (daemon *DockerDaemon) Images() ([]dockerTypes.ImageSummary, error) {

	return images(daemon.client, defaultImageListOptions)

}

//RunImage creates a container based on the given image and runs the given command
//Kind of like running "docker run $image $command" from the command line.
func (daemon *DockerDaemon) RunImage(image dockerTypes.ImageSummary, command string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	var imageName string
	if len(image.RepoTags) > 0 {
		imageName = image.RepoTags[0]
	} else if len(image.RepoDigests) > 0 {
		imageName = image.RepoDigests[0]

	} else {
		return pkgError.New("Cannot run image, image has no tag or digest")
	}
	runCommand := strings.Split(command, " ")

	cc := &container.Config{
		Image: imageName,
		Cmd:   strslice.StrSlice(runCommand)}
	cCreated, err := daemon.client.ContainerCreate(ctx, cc, nil, nil, "")

	if err != nil {
		return pkgError.Wrap(err, fmt.Sprintf("Cannot create container for image %s", imageName))
	}

	if err := daemon.client.ContainerStart(ctx, cCreated.ID, dockerTypes.ContainerStartOptions{}); err != nil {
		return pkgError.Wrap(err, fmt.Sprintf("Cannot start container %s for image %s", cCreated.ID, imageName))

	}
	return nil
}
