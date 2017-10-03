package app

import (
	"fmt"
	"sync"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/json"
	"github.com/nsf/termbox-go"
)

type commandToExecute struct {
	command   docker.Command
	container *docker.Container
}
type containersScreenEventHandler struct {
	baseEventHandler
}

func (h *containersScreenEventHandler) widget() appui.EventableWidget {
	return h.dry.widgetRegistry.ContainerList
}

func (h *containersScreenEventHandler) handle(event termbox.Event) {
	focus, handled := handleKey(h, event.Key)
	if !handled {
		focus, handled = handleCharacter(h, event.Ch)
	}
	if handled {
		h.setFocus(focus)
		if h.hasFocus() {
			refreshScreen()
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
		if err := dry.dockerDaemon.RestartContainer(id); err != nil {
			dry.appmessage(
				fmt.Sprintf("Error restarting container %s, err: %s", id, err.Error()))
		}
	case docker.STOP:
		if err := dry.dockerDaemon.StopContainer(id); err != nil {
			dry.appmessage(
				fmt.Sprintf("Error stopping container %s, err: %s", id, err.Error()))
		}
	case docker.LOGS:
		logs := h.dry.dockerDaemon.Logs(id)
		focus = false
		go appui.Stream(h.screen, logs, h.eventChan, h.closeViewChan)
	case docker.RM:
		dry.dockerDaemon.Rm(id)
	case docker.STATS:
		c := dry.dockerDaemon.ContainerByID(id)
		if c == nil || !docker.IsContainerRunning(c) {
			dry.appmessage(
				fmt.Sprintf("Container with id %s not found or is not running", id))
		} else {
			statsChan := dry.dockerDaemon.OpenChannel(c)
			focus = false
			go statsScreen(command.container, statsChan, screen, h.eventChan, h.closeViewChan)
		}

	case docker.INSPECT:
		container, err := h.dry.dockerDaemon.Inspect(id)
		if err == nil {
			go func() {
				defer func() {
					h.closeViewChan <- struct{}{}
				}()
				v, err := json.NewViewer(
					h.screen,
					appui.DryTheme,
					container)
				if err != nil {
					dry.appmessage(
						fmt.Sprintf("Error inspecting container: %s", err.Error()))
					return
				}
				v.Focus(h.eventChan)
			}()
			focus = false
		}
	case docker.HISTORY:
		history, err := dry.dockerDaemon.History(command.container.ImageID)

		if err == nil {
			focus = false
			renderer := appui.NewDockerImageHistoryRenderer(history)

			go appui.Less(renderer, screen, h.eventChan, h.closeViewChan)
		} else {
			dry.appmessage(
				fmt.Sprintf("Error showing image history: %s", err.Error()))
		}
	}
	if focus {
		h.closeViewChan <- struct{}{}
	}
}

func handleCharacter(h *containersScreenEventHandler, key rune) (focus, handled bool) {
	focus = true
	handled = false
	dry := h.dry
	switch key {
	case 'e', 'E': //remove
		handled = true
		if err := h.widget().OnEvent(
			func(id string) error {
				h.dry.appmessage("Removing container " + id)

				container := dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				//Since a command is created the focus is handled by handleCommand
				//Fixes #24
				focus = false
				h.handleCommand(commandToExecute{
					docker.RM,
					container,
				})
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error removing the container: " + err.Error())
		}

	case 'i', 'I': //inspect
		handled = true
		if err := h.widget().OnEvent(
			func(id string) error {
				container := dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				//Since a command is created the focus is handled by handleCommand
				//Fixes #24
				focus = false
				h.handleCommand(commandToExecute{
					docker.INSPECT,
					container,
				})
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error inspecting the container: " + err.Error())
		}

	case 'l', 'L': //logs
		handled = true
		if err := h.widget().OnEvent(
			func(id string) error {
				container := dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				//Since a command is created the focus is handled by handleCommand
				//Fixes #24
				focus = false
				h.handleCommand(commandToExecute{
					docker.LOGS,
					container,
				})
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error showing logs: " + err.Error())
		}
	case 's', 'S': //stats
		handled = true
		if err := h.widget().OnEvent(
			func(id string) error {
				container := dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				//Since a command is created the focus is handled by handleCommand
				//Fixes #24
				focus = false
				h.handleCommand(commandToExecute{
					docker.STATS,
					container,
				})
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error showing stats: " + err.Error())
		}
	}
	return focus, handled
}

func handleKey(h *containersScreenEventHandler, key termbox.Key) (bool, bool) {
	focus := true
	handled := true
	cursor := h.screen.Cursor
	switch key {
	case termbox.KeyF1: //sort
		h.dry.widgetRegistry.ContainerList.Sort()
	case termbox.KeyF2: //show all containers
		cursor.Reset()
		h.dry.widgetRegistry.ContainerList.ToggleShowAllContainers()

	case termbox.KeyF3: //filter containers
		if _, err := appui.ReadLine("Show containers named (leave empty to remove the filter) >>> "); err == nil {
			//TODO filter
			//h.dry.SetContainerFilter(filter)
		}
		h.screen.ClearAndFlush()
	case termbox.KeyF5: // refresh
		h.dry.appmessage("Refreshing container list")
		h.dry.dockerDaemon.Refresh(func(e error) {
			if e == nil {
				h.widget().Unmount()
				refreshScreen()
			} else {
				h.dry.appmessage("There was an error refreshing: " + e.Error())
			}
		})
	case termbox.KeyCtrlE: //remove all stopped
		if confirmation, err := appui.ReadLine("All stopped containers will be removed. Do you want to continue? (y/N) "); err == nil {
			h.screen.ClearAndFlush()
			if confirmation == "Y" || confirmation == "y" {
				h.dry.RemoveAllStoppedContainers()
			}
		}
	case termbox.KeyCtrlK: //kill
		if err := h.widget().OnEvent(
			func(id string) error {
				h.dry.appmessage("Killing container " + id)
				return h.dry.dockerDaemon.Kill(id)
			}); err != nil {
			h.dry.appmessage("There was an error killing the container: " + err.Error())
		}
	case termbox.KeyCtrlR: //start
		if err := h.widget().OnEvent(
			func(id string) error {
				h.dry.appmessage("Restarting container " + id)

				return h.dry.dockerDaemon.RestartContainer(id)
			}); err != nil {
			h.dry.appmessage("There was an error refreshing: " + err.Error())
		}
	case termbox.KeyCtrlT: //stop
		if err := h.widget().OnEvent(
			func(id string) error {
				h.dry.appmessage("Stopping container " + id)
				return h.dry.dockerDaemon.StopContainer(id)
			}); err != nil {
			h.dry.appmessage("There was an error killing the container: " + err.Error())
		}
	case termbox.KeyEnter: //Container menu
		showMenu := func(id string) error {
			container := h.dry.dockerDaemon.ContainerByID(id)
			go showContainerOptions(container, h)
			return nil
		}
		if err := h.widget().OnEvent(showMenu); err == nil {
			focus = false
		}

	default: //Not handled
		handled = false
	}
	return focus, handled
}

//statsScreen shows container stats on the screen
//TODO move to appui
func statsScreen(container *docker.Container, stats *docker.StatsChannel, screen *ui.Screen, keyboardQueue chan termbox.Event, closeView chan<- struct{}) {
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

	info, infoLines := appui.NewContainerInfo(container)
	screen.Render(1, info)

	var mutex = &sync.Mutex{}
	screen.Flush()

	s := stats.Stats

	header := appui.NewMonitorTableHeader()
	header.SetX(0)
	header.SetWidth(ui.ActiveScreen.Dimensions.Width)
	header.SetY(infoLines + 2)

	statsRow := appui.NewContainerStatsRow(container, header)
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
				statsRow.Update(container, stat)
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
func showContainerOptions(container *docker.Container, h *containersScreenEventHandler) {
	screen := h.screen
	selectedContainer := screen.Cursor.Position()
	keyboardQueue := h.eventChan
	closeView := h.closeViewChan
	//TODO handle error
	if container != nil {
		screen.Clear()
		screen.Sync()
		screen.Cursor.Reset()

		info, infoLines := appui.NewContainerInfo(container)
		screen.Cursor.Max(infoLines)
		screen.RenderLineWithBackGround(0, ui.ActiveScreen.Dimensions.Height-1, commandsMenuBar, appui.DryTheme.Footer)
		screen.Render(1, info)
		l := appui.NewContainerCommands(container,
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
