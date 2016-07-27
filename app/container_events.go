package app

import (
	"io"
	"sync"

	termui "github.com/gizak/termui"
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

func (h containersScreenEventHandler) handle(event termbox.Event) (refresh bool, focus bool) {
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
		refresh = true
	case termbox.KeyArrowDown: // cursor down
		cursor.ScrollCursorDown()
		refresh = true
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
		/*if cursorPos >= 0 {
			dry.Inspect(cursorPos)
			focus = false
			go less(dry, screen, h.keyboardQueueForView, h.closeView)
		}*/
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
				stats, done, err := dry.StatsAt(cursorPos)
				if err == nil {
					focus = false
					go statsScreen(screen, h.keyboardQueueForView, stats, h.closeView, done)
				}
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
	return (refresh || dry.Changed()), focus
}

//statsScreen shows container stats on the screen
func statsScreen(screen *ui.Screen, keyboardQueue chan termbox.Event, stats <-chan *docker.Stats, closeView chan<- struct{}, done chan<- struct{}) {
	screen.Clear()
	v := ui.NewMarkupView("", 0, 0, screen.Width, screen.Height, false)

	var mutex = &sync.Mutex{}
	err := v.Render()
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
	closeView <- struct{}{}
	close(done)
}

//statsScreen shows container stats on the screen
func showContainerOptions(dry *Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, closeView chan<- struct{}) {
	l := termui.NewList()
	//TODO handle error
	container, _ := dry.ContainerAt(screen.Cursor.Position())
	screen.Cursor.Reset()
	commandsLen := len(docker.CommandDescriptions)
	commands := make([]string, commandsLen)
	copy(commands, docker.CommandDescriptions)
	commands[0] = replaceAtIndex(
		commands[0],
		appui.RightArrow,
		0)
	l.Items = commands
	l.BorderLabel = " Container " + container.Names[0] + " commands "
	l.BorderLabelFg = termui.ColorBlue
	l.Height = screen.Height - appui.MainScreenHeaderSize - appui.MainScreenFooterSize
	l.Width = screen.Width
	l.X = 0
	l.Y = appui.MainScreenHeaderSize

	screen.RenderBufferer(l)
	screen.Flush()

	refreshChan := make(chan struct{}, 1)

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
			copy(commands, docker.CommandDescriptions)
			commands[screen.Cursor.Position()] = replaceAtIndex(
				commands[screen.Cursor.Position()],
				appui.RightArrow,
				0)
			screen.RenderBufferer(l)
			screen.Flush()
		}
	}

	//cleanup before exiting, the screen is cleared
	screen.Clear()
	screen.Sync()
	screen.Cursor.Reset()
	closeView <- struct{}{}
}

func replaceAtIndex(str string, replacement string, index int) string {
	return str[:index] + replacement + str[index+1:]
}
