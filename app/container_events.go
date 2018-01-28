package app

import (
	"fmt"
	"sync"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/json"
	termbox "github.com/nsf/termbox-go"
)

type commandToExecute struct {
	command   docker.Command
	container *docker.Container
}
type containersScreenEventHandler struct {
	baseEventHandler
}

func (h *containersScreenEventHandler) widget() appui.AppWidget {
	return h.dry.widgetRegistry.ContainerList
}

func (h *containersScreenEventHandler) handle(event termbox.Event) {
	if h.forwardingEvents {
		h.eventChan <- event
		return
	}
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

	closeView := true
	dry := h.dry
	screen := h.screen

	id := command.container.ID

	switch command.command {
	case docker.KILL:
		go func() {
			dry.actionMessage(id, "Killing")
			err := dry.dockerDaemon.Kill(id)
			if err == nil {
				dry.actionMessage(id, "killed")
			} else {
				dry.errorMessage(id, "killing", err)
			}
		}()
	case docker.RESTART:
		go func() {
			if err := dry.dockerDaemon.RestartContainer(id); err != nil {
				dry.appmessage(
					fmt.Sprintf("Error restarting container %s, err: %s", id, err.Error()))
			}
		}()
	case docker.STOP:
		go func() {
			if err := dry.dockerDaemon.StopContainer(id); err != nil {
				dry.appmessage(
					fmt.Sprintf("Error stopping container %s, err: %s", id, err.Error()))
			}
		}()
	case docker.LOGS:
		closeView = false
		h.setForwardEvents(true)
		prompt := logsPrompt()
		dry.widgetRegistry.add(prompt)
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			dry.widgetRegistry.remove(prompt)
			since, canceled := prompt.Text()

			if canceled {
				h.setForwardEvents(false)
				return
			}

			logs, err := h.dry.dockerDaemon.Logs(id, since)
			if err == nil {
				appui.Stream(logs, h.eventChan, func() {
					h.setForwardEvents(false)
					h.dry.SetViewMode(Main)
					h.closeViewChan <- struct{}{}
				})
			} else {
				h.dry.appmessage("Error showing container logs: " + err.Error())
				h.setForwardEvents(false)
			}
		}()
	case docker.RM:
		go func() {
			dry.actionMessage(id, "Removing")
			err := dry.dockerDaemon.Rm(id)
			if err == nil {
				dry.actionMessage(id, "removed")
			} else {
				dry.errorMessage(id, "removing", err)
			}
		}()

	case docker.STATS:
		c := dry.dockerDaemon.ContainerByID(id)
		if c == nil || !docker.IsContainerRunning(c) {
			dry.appmessage(
				fmt.Sprintf("Container with id %s not found or is not running", id))
		} else {
			statsChan := dry.dockerDaemon.OpenChannel(c)
			closeView = false
			go statsScreen(command.container, statsChan, screen, h.eventChan, func() {
				h.setForwardEvents(false)
				h.dry.SetViewMode(Main)
				h.closeViewChan <- struct{}{}
			})
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
			closeView = false
		}
	case docker.HISTORY:
		history, err := dry.dockerDaemon.History(command.container.ImageID)

		if err == nil {
			closeView = false
			renderer := appui.NewDockerImageHistoryRenderer(history)

			go appui.Less(renderer, screen, h.eventChan, h.closeViewChan)
		} else {
			dry.appmessage(
				fmt.Sprintf("Error showing image history: %s", err.Error()))
		}
	}
	if closeView {
		h.closeViewChan <- struct{}{}
	}
}

func handleCharacter(h *containersScreenEventHandler, key rune) (focus, handled bool) {
	focus = true
	handled = false
	dry := h.dry
	switch key {

	case '%': //filter containers
		handled = true
		showFilterInput(h)
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
		h.widget().Sort()
	case termbox.KeyF2: //show all containers
		cursor.Reset()
		h.dry.widgetRegistry.ContainerList.ToggleShowAllContainers()
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
			h.screen.Cursor.Reset()
			h.dry.ShowContainerMenu(id)
			return refreshScreen()
		}
		if err := h.widget().OnEvent(showMenu); err != nil {
			h.dry.appmessage(err.Error())
		}

	default: //Not handled
		handled = false
	}

	return focus, handled
}

//statsScreen shows container stats on the screen
//TODO move to appui
func statsScreen(container *docker.Container, stats *docker.StatsChannel, screen *ui.Screen, keyboardQueue chan termbox.Event, closeCallback func()) {
	defer closeCallback()
	screen.Clear()

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
