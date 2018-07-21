package app

import (
	"fmt"
	"sync"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

type commandRunner struct {
	command   docker.Command
	container *docker.Container
}
type containersScreenEventHandler struct {
	baseEventHandler
	widget appui.AppWidget
}

func (h *containersScreenEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	if h.forwardingEvents() {
		h.eventChan <- event
		return
	}
	handled := h.handleKey(event.Key, f)

	if !handled {
		handled = h.handleCharacter(event.Ch, f)
	}
	if handled {
		refreshScreen()
	} else {
		h.baseEventHandler.handle(event, f)
	}
}

func (h *containersScreenEventHandler) handleCommand(command commandRunner, f func(eventHandler)) {
	dry := h.dry
	screen := h.screen
	id := command.container.ID

	switch command.command {
	case docker.KILL:
		prompt := appui.NewPrompt(
			fmt.Sprintf("Do you want to kill container %s? (y/N)", id))
		widgets.add(prompt)
		h.setForwardEvents(true)

		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			h.setForwardEvents(false)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {

				return
			}

			dry.actionMessage(id, "Killing")
			err := dry.dockerDaemon.Kill(id)
			if err == nil {
				dry.actionMessage(id, "killed")
			} else {
				dry.errorMessage(id, "killing", err)
			}

		}()

	case docker.RESTART:
		prompt := appui.NewPrompt(
			fmt.Sprintf("Do you want to restart container %s? (y/N)", id))
		widgets.add(prompt)
		h.setForwardEvents(true)

		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			h.setForwardEvents(false)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {

				return
			}

			if err := dry.dockerDaemon.RestartContainer(id); err != nil {
				dry.appmessage(
					fmt.Sprintf("Error restarting container %s, err: %s", id, err.Error()))
			}

		}()

	case docker.STOP:
		prompt := appui.NewPrompt(
			fmt.Sprintf("Do you want to remove container %s? (y/N)", id))
		widgets.add(prompt)
		h.setForwardEvents(true)

		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			h.setForwardEvents(false)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {

				return
			}

			if err := dry.dockerDaemon.StopContainer(id); err != nil {
				dry.appmessage(
					fmt.Sprintf("Error stopping container %s, err: %s", id, err.Error()))
			}

		}()

	case docker.LOGS:
		prompt := logsPrompt()
		widgets.add(prompt)
		h.setForwardEvents(true)
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			widgets.remove(prompt)
			since, canceled := prompt.Text()

			if canceled {
				h.setForwardEvents(false)
				return
			}

			logs, err := h.dry.dockerDaemon.Logs(id, since)
			if err == nil {
				appui.Stream(logs, h.eventChan, func() {
					h.dry.SetViewMode(Main)
					f(viewsToHandlers[Main])
					h.setForwardEvents(false)
					refreshScreen()
				})
			} else {
				h.dry.appmessage("Error showing container logs: " + err.Error())

			}
		}()
	case docker.RM:
		prompt := appui.NewPrompt(
			fmt.Sprintf("Do you want to remove container %s? (y/N)", id))
		widgets.add(prompt)
		h.setForwardEvents(true)

		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			h.setForwardEvents(false)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {

				return
			}

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
				fmt.Sprintf("Container with id %s not found or not running", id))
		} else {
			statsChan := dry.dockerDaemon.OpenChannel(c)
			h.setForwardEvents(true)
			h.dry.SetViewMode(NoView)
			go statsScreen(command.container, statsChan, screen, h.eventChan, func() {
				h.dry.SetViewMode(Main)
				f(viewsToHandlers[Main])
				h.setForwardEvents(false)
				refreshScreen()
			})
		}

	case docker.INSPECT:
		h.setForwardEvents(true)
		err := inspect(
			h.screen,
			h.eventChan,
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.Inspect(id)
			},
			func() {
				h.dry.SetViewMode(Main)
				h.setForwardEvents(false)
				f(h)
				refreshScreen()
			})(id)

		if err != nil {
			dry.appmessage(
				fmt.Sprintf("Error inspecting container: %s", err.Error()))
			return
		}

	case docker.HISTORY:
		history, err := dry.dockerDaemon.History(command.container.ImageID)

		if err == nil {

			renderer := appui.NewDockerImageHistoryRenderer(history)

			go appui.Less(renderer, screen, h.eventChan, func() {
				h.dry.SetViewMode(Main)
				f(viewsToHandlers[Main])
				h.setForwardEvents(false)
			})
		} else {
			dry.appmessage(
				fmt.Sprintf("Error showing image history: %s", err.Error()))
		}
	}
}

func (h *containersScreenEventHandler) handleCharacter(key rune, f func(eventHandler)) bool {
	handled := true
	dry := h.dry
	switch key {

	case '%': //filter containers
		h.setForwardEvents(true)
		applyFilter := func(filter string, canceled bool) {
			if !canceled {
				h.widget.Filter(filter)
			}
			h.setForwardEvents(false)
		}
		showFilterInput(newEventSource(h.eventChan), applyFilter)

	case 'e', 'E': //remove
		if err := h.widget.OnEvent(
			func(id string) error {
				container := dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				h.handleCommand(commandRunner{
					docker.RM,
					container,
				}, f)
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error removing the container: " + err.Error())
		}

	case 'i', 'I': //inspect
		if err := h.widget.OnEvent(
			func(id string) error {
				container := dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				h.handleCommand(commandRunner{
					docker.INSPECT,
					container,
				}, f)
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error inspecting the container: " + err.Error())
		}

	case 'l', 'L': //logs
		if err := h.widget.OnEvent(
			func(id string) error {
				container := dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				h.handleCommand(commandRunner{
					docker.LOGS,
					container,
				}, f)
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error showing logs: " + err.Error())
		}
	case 's', 'S': //stats
		if err := h.widget.OnEvent(
			func(id string) error {
				container := dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				h.handleCommand(commandRunner{
					docker.STATS,
					container,
				}, f)
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error showing stats: " + err.Error())
		}
	default:
		handled = false
	}
	return handled
}

func (h *containersScreenEventHandler) handleKey(key termbox.Key, f func(eventHandler)) bool {
	handled := true
	cursor := h.screen.Cursor
	switch key {
	case termbox.KeyF1: //sort
		h.widget.Sort()
	case termbox.KeyF2: //show all containers
		cursor.Reset()
		widgets.ContainerList.ToggleShowAllContainers()
	case termbox.KeyF5: // refresh
		h.dry.appmessage("Refreshing container list")
		h.dry.dockerDaemon.Refresh(func(e error) {
			if e == nil {
				h.widget.Unmount()
				refreshScreen()
			} else {
				h.dry.appmessage("There was an error refreshing: " + e.Error())
			}
		})
	case termbox.KeyCtrlE: //remove all stopped
		prompt := appui.NewPrompt(
			"All stopped containers will be removed. Do you want to continue? (y/N) ")
		widgets.add(prompt)
		h.setForwardEvents(true)

		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			h.setForwardEvents(false)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {
				return
			}
			go func() {
				h.dry.appmessage("<red>Removing all stopped containers</>")
				if count, err := h.dry.dockerDaemon.RemoveAllStoppedContainers(); err == nil {
					h.dry.appmessage(fmt.Sprintf("<red>Removed %d stopped containers</>", count))
				} else {
					h.dry.appmessage(
						fmt.Sprintf(
							"<red>Error removing all stopped containers: %s</>", err.Error()))
				}
				refreshScreen()
			}()
		}()

	case termbox.KeyCtrlK: //kill
		if err := h.widget.OnEvent(
			func(id string) error {
				container := h.dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				h.handleCommand(commandRunner{
					docker.KILL,
					container,
				}, f)
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error killing container: " + err.Error())
		}
	case termbox.KeyCtrlR: //start
		if err := h.widget.OnEvent(
			func(id string) error {
				container := h.dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				h.handleCommand(commandRunner{
					docker.RESTART,
					container,
				}, f)
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error restarting: " + err.Error())
		}
	case termbox.KeyCtrlT: //stop
		if err := h.widget.OnEvent(
			func(id string) error {
				container := h.dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				h.handleCommand(commandRunner{
					docker.STOP,
					container,
				}, f)
				return nil
			}); err != nil {
			h.dry.appmessage("There was an error stopping container: " + err.Error())
		}
	case termbox.KeyEnter: //Container menu
		showMenu := func(id string) error {
			h.screen.Cursor.Reset()
			widgets.ContainerMenu.ForContainer(id)
			h.dry.SetViewMode(ContainerMenu)
			f(viewsToHandlers[ContainerMenu])
			return refreshScreen()
		}
		if err := h.widget.OnEvent(showMenu); err != nil {
			h.dry.appmessage(err.Error())
		}

	default: //Not handled
		handled = false
	}

	return handled
}

//statsScreen shows container stats on the screen
//TODO move to appui
func statsScreen(container *docker.Container, stats *docker.StatsChannel, screen *ui.Screen, events <-chan termbox.Event, closeCallback func()) {
	if !docker.IsContainerRunning(container) {
		return
	}
	defer closeCallback()
	screen.ClearAndFlush()

	info, infoLines := appui.NewContainerInfo(container)
	screen.Render(1, info)

	var mutex sync.Mutex
	s := stats.Stats

	header := appui.NewMonitorTableHeader()
	header.SetX(0)
	header.SetWidth(ui.ActiveScreen.Dimensions.Width)
	header.SetY(infoLines + 3)

	statsRow := appui.NewContainerStatsRow(container, header)
	statsRow.SetX(0)
	statsRow.SetY(header.Y + 1)
	statsRow.SetWidth(ui.ActiveScreen.Dimensions.Width)

loop:
	for {
		select {
		case event := <-events:
			if event.Type == termbox.EventKey && event.Key == termbox.KeyEsc {
				//the lock is acquired before breaking the loop
				mutex.Lock()
				s = nil
			}

		case stat, ok := <-s:
			{
				if !ok {
					break loop
				}
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
