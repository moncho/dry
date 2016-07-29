package appui

import (
	"fmt"

	"github.com/docker/engine-api/types"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
)

//ContainerCommandList is a Bufferer holding the list of container commands
type ContainerCommandList struct {
	List     *termui.List
	Commands []string
}

//NewContainerCommands creates a Bufferer with the list of container commands
func NewContainerCommands(container types.Container, height, width int) *ContainerCommandList {
	l := termui.NewList()

	shortID := docker.TruncateID(container.ID)
	commandsLen := len(docker.CommandDescriptions)
	commands := make([]string, commandsLen)
	l.Items = commands
	l.BorderLabel =
		fmt.Sprintf(" %s - %s ", container.Names[0], shortID)
	l.BorderLabelFg = termui.ColorYellow
	l.Height = height
	l.Width = width
	l.PaddingTop = 2
	l.PaddingLeft = 2
	l.X = 0
	l.Y = MainScreenHeaderSize

	return &ContainerCommandList{l, commands}
}
