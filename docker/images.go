package docker

import (
	"context"
	"errors"
	"fmt"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

// History returns image history
func (daemon *DockerDaemon) History(id string) ([]image.HistoryResponseItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	res, err := daemon.client.ImageHistory(ctx, id)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// ImageByID returns the image with the given ID
func (daemon *DockerDaemon) ImageByID(id string) (image.Summary, error) {
	var result image.Summary
	images, err := daemon.Images()
	if err != nil {
		return result, err
	}
	for _, image := range images {
		if image.ID == id {
			return image, nil
		}
	}

	return result, fmt.Errorf("image %s not found", id)
}

// Images returns the list of Docker images
func (daemon *DockerDaemon) Images() ([]image.Summary, error) {
	res, err := images(daemon.client, defaultImageListOptions)
	if err != nil {
		return []image.Summary{}, err
	}
	return res.Items, nil
}

// RunImage creates a container based on the given image and runs the given command
// Kind of like running "docker run $image $command" from the command line.
func (daemon *DockerDaemon) RunImage(image image.Summary, command string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	var imageName string
	if len(image.RepoTags) > 0 {
		imageName = image.RepoTags[0]
	} else if len(image.RepoDigests) > 0 {
		imageName = image.RepoDigests[0]
	} else {
		return errors.New("run image: image has no tag or digest")
	}

	imageDetails, err := daemon.InspectImage(imageName)
	if err != nil {
		return fmt.Errorf("run image: inspect image %s: %w", imageName, err)
	}

	var exposedPorts map[string]struct{}
	if imageDetails.Config != nil && imageDetails.Config.ExposedPorts != nil {
		exposedPorts = imageDetails.Config.ExposedPorts
	}
	cc, hc, err := newCCB().image(imageName).command(command).ports(exposedPorts).build()
	if err != nil {
		return fmt.Errorf("run image: %w", err)
	}

	cCreated, err := daemon.client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:     &cc,
		HostConfig: &hc,
	})
	if err != nil {
		return fmt.Errorf("run image: create container for image %s: %w", imageName, err)
	}

	if _, err := daemon.client.ContainerStart(ctx, cCreated.ID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("run image: start container for image %s: %w", imageName, err)
	}
	return nil
}
