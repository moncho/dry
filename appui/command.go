package appui

import (
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//ContainerCommandList is a Bufferer holding the list of container commands
type ContainerCommandList struct {
	List     *termui.List
	Commands []string
}

//NewContainerCommands creates a Bufferer with the list of container commands
func NewContainerCommands(container *docker.Container, x, y, height, width int) *ContainerCommandList {
	l := ui.NewList(DryTheme)

	commandsLen := len(docker.CommandDescriptions)
	commands := make([]string, commandsLen)

	l.Items = commands
	l.Border = false
	l.BorderFg = termui.ColorBlue
	l.Height = len(commands) + 4 // 2 because of the top+bottom padding, 2 because of the borders
	l.Width = width / 2
	l.PaddingTop = 1
	l.PaddingBottom = 1
	l.PaddingLeft = 2
	l.X = x
	l.Y = y

	return &ContainerCommandList{l, commands}
}
