package app

import (
	"io"
	"sync"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type containersScreenEventHandler struct {
	dry                  *Dry
	screen               *ui.Screen
	keyboardQueueForView chan termbox.Event
	closeView            chan struct{}
}

func (h containersScreenEventHandler) handle(renderChan chan<- struct{}, event termbox.Event) (focus bool) {
	focus = true
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
		go less(dry, screen, h.keyboardQueueForView, h.closeView)
	case termbox.KeyF10: // docker info
		dry.ShowInfo()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.closeView)
	case termbox.KeyCtrlE: //remove all stopped
		dry.RemoveAllStoppedContainers()
	case termbox.KeyCtrlK: //kill
		if cursorPos >= 0 {
			dry.KillAt(cursorPos)
		}
	case termbox.KeyCtrlR: //start
		dry.RestartContainerAt(cursorPos)
	case termbox.KeyCtrlT: //stop
		dry.StopContainerAt(cursorPos)
	case termbox.KeyEnter: //inspect
		if cursorPos >= 0 {
			focus = false
			go showContainerOptions(dry, screen, h.keyboardQueueForView, h.closeView)
		}
	default: //Not handled
		handled = false
	}
	if !handled {
		switch event.Ch {
		case 's', 'S': //stats
			if cursorPos >= 0 {
				focus = false
				go statsScreen(screen, dry, h.keyboardQueueForView, h.closeView)
			}
		case 'i', 'I': //inspect
			if cursorPos >= 0 {
				dry.InspectAt(cursorPos)
				focus = false
				go less(dry, screen, h.keyboardQueueForView, h.closeView)
			}
		case 'l', 'L': //logs
			if cursorPos >= 0 {
				if logs, err := dry.LogsAt(cursorPos); err == nil {
					focus = false
					dry.ShowContainers()
					go stream(screen, logs, h.keyboardQueueForView, h.closeView)
				}
			}
		case '?', 'h', 'H': //help
			focus = false
			dry.ShowHelp()
			go less(dry, screen, h.keyboardQueueForView, h.closeView)
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

//statsScreen shows container stats on the screen
func statsScreen(screen *ui.Screen, dry *Dry, keyboardQueue chan termbox.Event, closeView chan<- struct{}) {
	defer func() {
		closeView <- struct{}{}
	}()
	screen.Clear()

	//TODO handle error
	container, _ := dry.ContainerAt(screen.Cursor.Position())
	if !docker.IsContainerRunning(container) {
		return
	}

	stats, done, err := dry.Stats(container.ID)
	if err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, err)
	}
	info, infoLines := appui.NewContainerInfo(container)
	screen.Render(1, info)
	v := ui.NewMarkupView("", 0, infoLines+1, screen.Width, screen.Height, false)

	var mutex = &sync.Mutex{}
	err = v.Render()
	if err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, err)
	}
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
					break loop
				}
			}
		case s := <-stats:
			{
				mutex.Lock()
				v.Clear()
				io.WriteString(v, appui.NewDockerStatsRenderer(s).Render())
				v.Render()
				screen.Flush()
				mutex.Unlock()
			}
		}
	}
	//cleanup before exiting, the screen is cleared and the lock released
	screen.Clear()
	screen.Sync()
	mutex.Unlock()
	close(done)
}

//statsScreen shows container stats on the screen
func showContainerOptions(dry *Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, closeView chan<- struct{}) {

	defer func() {
		closeView <- struct{}{}
	}()

	//TODO handle error
	container, _ := dry.ContainerAt(screen.Cursor.Position())
	screen.Clear()
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

	refreshChan <- struct{}{}

loop:
	for {
		select {
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				if event.Key == termbox.KeyEsc {
					refreshChan = nil
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
					command := docker.ContainerCommands[screen.Cursor.Position()]
					refreshChan = nil
					dry.runCommand(command.Command, container.ID)
					break loop
				}
			}
		case <-refreshChan:
			markSelectedCommand(l.Commands, screen.Cursor.Position())
			screen.RenderBufferer(l.List)
			screen.Flush()
		}
	}

	//cleanup before exiting, the screen is cleared
	screen.Clear()
	screen.Sync()
	screen.Cursor.Reset()
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
