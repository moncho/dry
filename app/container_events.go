package app

import (
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type commandToExecute struct {
	command   docker.Command
	container *types.Container
}
type containersScreenEventHandler struct {
	baseEventHandler
}

func (h *containersScreenEventHandler) handle(event termbox.Event) {
	focus := true
	dry := h.dry
	screen := h.screen
	cursor := screen.Cursor
	cursorPos := cursor.Position()
	//Controls if the event has been handled by the first¡ switch statement
	handled := true
	switch event.Key {
	case termbox.KeyF1: //sort
		dry.Sort()
	case termbox.KeyF2: //show all containers
		cursor.Reset()
		dry.ToggleShowAllContainers()
	case termbox.KeyF3: //filter containers
		if filter, err := appui.ReadLine("Show containers named (leave empty to remove the filter) >>> "); err == nil {
			dry.SetContainerFilter(filter)
		}
		screen.ClearAndFlush()
	case termbox.KeyCtrlE: //remove all stopped
		if confirmation, err := appui.ReadLine("All stopped containers will be removed. Do you want to continue? (y/N) "); err == nil {
			screen.ClearAndFlush()
			if confirmation == "Y" || confirmation == "y" {
				dry.RemoveAllStoppedContainers()
			}
		}
	case termbox.KeyCtrlK: //kill
		dry.KillAt(cursorPos)
	case termbox.KeyCtrlR: //start
		dry.RestartContainerAt(cursorPos)
	case termbox.KeyCtrlT: //stop
		dry.StopContainerAt(cursorPos)
	case termbox.KeyEnter: //inspect
		focus = false
		go showContainerOptions(h, dry, screen, h.keyboardQueueForView, h.closeViewChan)
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
					container,
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
						container,
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
						container,
					})
				}
			}
		case 's', 'S': //stats
			handled = true
			if cursorPos >= 0 {
				container := dry.ContainerAt(cursorPos)
				if container != nil {
					focus = false
					h.handleCommand(commandToExecute{
						docker.STATS,
						container,
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

func (h *containersScreenEventHandler) handleCommand(command commandToExecute) {
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

//statsScreen shows container stats on the screen
//TODO move to appui
func statsScreen(container *types.Container, screen *ui.Screen, dry *Dry, keyboardQueue chan termbox.Event, closeView chan<- struct{}) {
	closeViewOnExit := true
	screen.Clear()

	defer func() {
		if closeViewOnExit {
			closeView <- struct{}{}
		}
	}()

	if !docker.IsContainerRunning(container) {
		return
	}

	stats, err := dry.Stats(container.ID)
	if err != nil {
		closeViewOnExit = false
		ui.ShowErrorMessage(screen, keyboardQueue, closeView, err)
		return
	}
	info, infoLines := appui.NewContainerInfo(container)
	screen.Render(1, info)

	var mutex = &sync.Mutex{}
	screen.Flush()

	s := stats.Stats

	header := appui.NewMonitorTableHeader()
	header.SetX(0)
	header.SetWidth(ui.ActiveScreen.Dimensions.Width)
	header.SetY(infoLines + 2)

	statsRow := appui.NewContainerStatsRow(container)
	statsRow.SetX(0)
	statsRow.SetY(header.Y + 1)
	statsRow.SetWidth(ui.ActiveScreen.Dimensions.Width)

loop:
	for {
		select {
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				if event.Key == termbox.KeyEsc {
					//the lock is acquired before breaking the loop
					mutex.Lock()
					s = nil
				}
			}
		case stat := <-s:
			{
				mutex.Lock()
				statsRow.Update(stat)
				top, _ := appui.NewDockerTop(
					stat.ProcessList,
					0, statsRow.Y+2,
					ui.ActiveScreen.Dimensions.Height-infoLines-statsRow.GetHeight(),
					ui.ActiveScreen.Dimensions.Width)
				screen.RenderBufferer(
					header,
					top,
					statsRow)
				screen.Flush()
				mutex.Unlock()
			}
		}
		if s == nil {
			break loop
		}
	}
	//cleanup before exiting, the screen is cleared and the lock released
	screen.Clear()
	screen.Sync()
	mutex.Unlock()
	close(stats.Done)
}

//statsScreen shows container stats on the screen
func showContainerOptions(h *containersScreenEventHandler, dry *Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, closeView chan<- struct{}) {

	selectedContainer := screen.Cursor.Position()
	//TODO handle error
	container := dry.ContainerAt(selectedContainer)
	if container != nil {
		screen.Clear()
		screen.Sync()
		screen.Cursor.Reset()

		info, infoLines := appui.NewContainerInfo(container)
		screen.Cursor.Max(infoLines)
		screen.RenderLineWithBackGround(0, ui.ActiveScreen.Dimensions.Height-1, commandsMenuBar, appui.DryTheme.Footer)
		screen.Render(1, info)
		l := appui.NewContainerCommands(*container,
			0,
			infoLines+1,
			ui.ActiveScreen.Dimensions.Height-appui.MainScreenFooterSize-infoLines-1,
			ui.ActiveScreen.Dimensions.Width)
		commandsLen := len(l.Commands)
		refreshChan := make(chan struct{}, 1)
		var command docker.CommandDescription
		refreshChan <- struct{}{}

		go func() {
			for {
				_, ok := <-refreshChan
				if ok {
					markSelectedCommand(l.Commands, screen.Cursor.Position())
					screen.RenderBufferer(l.List)
					screen.Flush()
				} else {
					return
				}
			}
		}()

	loop:
		for event := range keyboardQueue {
			switch event.Type {
			case termbox.EventKey:
				if event.Key == termbox.KeyEsc {
					close(refreshChan)
					break loop
				} else if event.Key == termbox.KeyArrowUp { //cursor up
					if screen.Cursor.Position() > 0 {
						screen.Cursor.ScrollCursorUp()
						refreshChan <- struct{}{}
					}
				} else if event.Key == termbox.KeyArrowDown { // cursor down
					if screen.Cursor.Position() < commandsLen-1 {
						screen.Cursor.ScrollCursorDown()
						refreshChan <- struct{}{}
					}
				} else if event.Key == termbox.KeyEnter { // execute command
					command = docker.ContainerCommands[screen.Cursor.Position()]
					close(refreshChan)
					break loop
				}
			}
		}

		screen.Clear()
		screen.Sync()
		screen.Cursor.ScrollTo(selectedContainer)

		if (docker.CommandDescription{}) != command {
			h.handleCommand(
				commandToExecute{
					command.Command,
					container,
				})
		} else {
			//view is closed here if there is not a command to execute
			closeView <- struct{}{}
		}
	} else {
		//view is closed here if there is not a command to execute
		closeView <- struct{}{}
	}
}

//adds an arrow character before the command description on the given index
func markSelectedCommand(commands []string, index int) {
	copy(commands, docker.CommandDescriptions)
	commands[index] = replaceAtIndex(
		commands[index],
		appui.RightArrow,
		0)
}

func replaceAtIndex(str string, replacement string, index int) string {
	return str[:index] + replacement + str[index+1:]
}
