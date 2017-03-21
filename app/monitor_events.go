package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/nsf/termbox-go"
)

type monitorScreenEventHandler struct {
	baseEventHandler
}

func (h *monitorScreenEventHandler) handle(event termbox.Event) {
	focus := true
	dry := h.dry
	screen := h.screen
	cursor := screen.Cursor
	cursorPos := cursor.Position()
	//Controls if the event has been handled by the first switch statement
	handled := true
	switch event.Key {
	case termbox.KeyF1: //sort
		dry.Sort()
	case termbox.KeyF2: //show all containers
		cursor.Reset()
		dry.ToggleShowAllContainers()
	case termbox.KeyCtrlK: //kill
		dry.KillAt(cursorPos)
	case termbox.KeyCtrlR: //start
		dry.RestartContainerAt(cursorPos)
	case termbox.KeyCtrlT: //stop
		dry.StopContainerAt(cursorPos)
	default: //Not handled
		handled = false
	}
	if !handled {

		switch event.Ch {
		case 'e', 'E': //remove
			handled = true

			container := dry.ContainerAt(cursorPos)
			if container != nil {
				//Since a command is created the focus is handled by handleCommand
				//Fixes #24
				focus = false
				h.handleCommand(commandToExecute{
					docker.RM,
					*container,
				})
			}
		case 'i', 'I': //inspect
			handled = true

			if cursorPos >= 0 {
				container := dry.ContainerAt(cursorPos)
				if container != nil {
					focus = false

					h.handleCommand(commandToExecute{
						docker.INSPECT,
						*container,
					})
				}
			}
		case 'l', 'L': //logs
			handled = true

			if cursorPos >= 0 {
				container := dry.ContainerAt(cursorPos)
				if container != nil {
					focus = false

					h.handleCommand(commandToExecute{
						docker.LOGS,
						*container,
					})
				}
			}

		}
	}
	if handled {
		h.setFocus(focus)
		if h.hasFocus() {
			h.renderChan <- struct{}{}
		}
	} else {
		h.baseEventHandler.handle(event)
	}
}

func (h *monitorScreenEventHandler) handleCommand(command commandToExecute) {
	focus := true
	dry := h.dry
	screen := h.screen

	id := command.container.ID

	switch command.command {
	case docker.KILL:
		dry.Kill(id)
	case docker.RESTART:
		dry.RestartContainer(id)
	case docker.STOP:
		dry.StopContainer(id)
	case docker.LOGS:
		if logs, err := dry.Logs(id); err == nil {
			focus = false
			go appui.Stream(screen, logs, h.keyboardQueueForView, h.closeViewChan)
		}
	case docker.RM:
		dry.Rm(id)
	case docker.STATS:
		focus = false
		go statsScreen(command.container, screen, dry, h.keyboardQueueForView, h.closeViewChan)
	case docker.INSPECT:
		dry.Inspect(id)
		focus = false
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.closeViewChan)
	case docker.HISTORY:
		dry.History(command.container.ImageID)
		focus = false
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.closeViewChan)
	}
	if focus {
		h.closeViewChan <- struct{}{}
	}
}
