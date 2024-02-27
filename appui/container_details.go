package appui

import (
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	drytermui "github.com/moncho/dry/ui/termui"
)

// ContainerDetailsWidget shows service information
type ContainerDetailsWidget struct {
	drytermui.SizableBufferer
}

// NewContainerDetailsWidget creates ContainerDetailsWidget with information about the service with the given ID
func NewContainerDetailsWidget(container *docker.Container, y int) *ContainerDetailsWidget {
	info, lines := NewContainerInfo(container)

	cInfo := drytermui.NewParFromMarkupText(DryTheme, info)
	cInfo.Y = y
	cInfo.Height = lines + 1
	cInfo.BorderLeft = false
	cInfo.BorderRight = false
	cInfo.BorderTop = false

	cInfo.Bg = termui.Attribute(DryTheme.Bg)
	cInfo.BorderBg = termui.Attribute(DryTheme.Bg)
	cInfo.BorderFg = termui.Attribute(DryTheme.Footer)
	cInfo.TextBgColor = termui.Attribute(DryTheme.Bg)

	return &ContainerDetailsWidget{cInfo}
}
