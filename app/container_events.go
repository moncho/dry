package app

import (
	"sync"

	"github.com/docker/engine-api/types"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type commandToExecute struct {
	command   docker.Command
	container types.Container
}
type containersScreenEventHandler struct {
	dry                  *Dry
	screen               *ui.Screen
	keyboardQueueForView chan termbox.Event
	closeView            chan struct{}
}

func (h containersScreenEventHandler) handle(renderChan chan<- struct{}, event termbox.Event) bool {
	focus := true
	dry := h.dry
	screen := h.screen
	cursor := screen.Cursor
	cursorPos := cursor.Position()
	//Controls if the event has been handled by the first switch statement
	handled := true
	switch event.Key {
	case termbox.KeyArrowUp: //cursor up
		cursor.ScrollCursorUp()
	case termbox.KeyArrowDown: // cursor down
		cursor.ScrollCursorDown()
	case termbox.KeyF1: //sort
		dry.Sort()
	case termbox.KeyF2: //show all containers
		cursor.Reset()
		dry.ToggleShowAllContainers()
	case termbox.KeyF5: // refresh
		dry.Refresh()
	case termbox.KeyF9: // docker events
		dry.ShowDockerEvents()
		focus = false
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.closeView)
	case termbox.KeyF10: // docker info
		dry.ShowInfo()
		focus = false
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.closeView)
	case termbox.KeyCtrlE: //remove all stopped
		dry.RemoveAllStoppedContainers()
	case termbox.KeyCtrlK: //kill
		dry.KillAt(cursorPos)
	case termbox.KeyCtrlR: //start
		dry.RestartContainerAt(cursorPos)
	case termbox.KeyCtrlT: //stop
		dry.StopContainerAt(cursorPos)
	case termbox.KeyEnter: //inspect
		if cursorPos >= 0 {
			focus = false
			go showContainerOptions(h, dry, screen, h.keyboardQueueForView, h.closeView)
		}
	default: //Not handled
		handled = false
	}
	if !handled {
		switch event.Ch {
		case 's', 'S': //stats
			if cursorPos >= 0 {
				container, err := dry.ContainerAt(cursorPos)
				focus = false
				if err == nil {
					h.handleCommand(commandToExecute{
						docker.STATS,
						container,
					})
				} else {
					ui.ShowErrorMessage(screen, h.keyboardQueueForView, h.closeView, err)
				}
			}
		case 'i', 'I': //inspect
			if cursorPos >= 0 {
				focus = false
				container, err := dry.ContainerAt(cursorPos)
				if err == nil {
					h.handleCommand(commandToExecute{
						docker.INSPECT,
						container,
					})
				} else {
					ui.ShowErrorMessage(screen, h.keyboardQueueForView, h.closeView, err)
				}
			}
		case 'l', 'L': //logs
			if cursorPos >= 0 {
				focus = false
				container, err := dry.ContainerAt(cursorPos)
				if err == nil {
					h.handleCommand(commandToExecute{
						docker.LOGS,
						container,
					})
				} else {
					ui.ShowErrorMessage(screen, h.keyboardQueueForView, h.closeView, err)
				}
			}
		case '?', 'h', 'H': //help
			focus = false
			dry.ShowHelp()
			go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.closeView)
		case '2':
			cursor.Reset()
			dry.ShowImages()
		case '3':
			cursor.Reset()
			dry.ShowNetworks()
		case 'e', 'E': //remove
			if cursorPos >= 0 {
				dry.RmAt(cursorPos)
				cursor.ScrollCursorDown()
			}
		}
	}
	if focus {
		renderChan <- struct{}{}
	}
	return focus
}

func (h containersScreenEventHandler) handleCommand(command commandToExecute) {
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
			go appui.Stream(screen, logs, h.keyboardQueueForView, h.closeView)
		}
	case docker.STATS:
		focus = false
		go statsScreen(command.container, screen, dry, h.keyboardQueueForView, h.closeView)
	case docker.INSPECT:
		dry.Inspect(id)
		focus = false
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.closeView)
	}
	if focus {
		h.closeView <- struct{}{}
	}
}

//statsScreen shows container stats on the screen
//TODO move to appui
func statsScreen(container types.Container, screen *ui.Screen, dry *Dry, keyboardQueue chan termbox.Event, closeView chan<- struct{}) {
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

	stats, done, err := dry.Stats(container.ID)
	if err != nil {
		closeViewOnExit = false
		ui.ShowErrorMessage(screen, keyboardQueue, closeView, err)
		return
	}
	info, infoLines := appui.NewContainerInfo(container)
	screen.Render(1, info)

	var mutex = &sync.Mutex{}
	screen.Flush()

loop:
	for {
		select {
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				if event.Key == termbox.KeyEsc {
					//the lock is acquired before breaking the loop
					mutex.Lock()
					stats = nil
				}
			}
		case s := <-stats:
			{
				mutex.Lock()
				screen.RenderBufferer(
					appui.NewDockerStatsBufferer(
						s, 0, infoLines+3, screen.Height, screen.Width)...)
				screen.Flush()
				mutex.Unlock()
			}
		}
		if stats == nil {
			break loop
		}
	}
	//cleanup before exiting, the screen is cleared and the lock released
	screen.Clear()
	screen.Sync()
	mutex.Unlock()
	close(done)
}

//statsScreen shows container stats on the screen
func showContainerOptions(h containersScreenEventHandler, dry *Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, closeView chan<- struct{}) {

	//TODO handle error
	container, _ := dry.ContainerAt(screen.Cursor.Position())
	screen.Clear()
	screen.Sync()
	screen.Cursor.Reset()

	info, infoLines := appui.NewContainerInfo(container)
	screen.RenderLineWithBackGround(0, screen.Height-1, commandsMenuBar, ui.MenuBarBackgroundColor)
	screen.Render(1, info)
	l := appui.NewContainerCommands(container,
		0,
		infoLines+1,
		screen.Height-appui.MainScreenFooterSize-infoLines-1,
		screen.Width)
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
	for {
		select {
		case event := <-keyboardQueue:
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
	}

	screen.Clear()
	screen.Sync()
	screen.Cursor.Reset()

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
