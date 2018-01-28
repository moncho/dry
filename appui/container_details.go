package appui

import (
	"github.com/moncho/dry/docker"
	drytermui "github.com/moncho/dry/ui/termui"
)

//ContainerDetailsWidget shows service information
type ContainerDetailsWidget struct {
	container *docker.Container
	drytermui.SizableBufferer
}

//NewContainerDetailsWidget creates ContainerDetailsWidget with information about the service with the given ID
func NewContainerDetailsWidget(container *docker.Container, y int) *ContainerDetailsWidget {
	w := ContainerDetailsWidget{}

	return &w
}
