package appui

import (
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
func NewContainerCommands(container types.Container, x, y, height, width int) *ContainerCommandList {
	l := termui.NewList()

	//shortID := docker.TruncateID(container.ID)
	commandsLen := len(docker.CommandDescriptions)
	commands := make([]string, commandsLen)
	l.Items = commands
	l.Border = false
	//l.BorderLabel =
	//	fmt.Sprintf(" %s - %s ", container.Names[0], shortID)
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
