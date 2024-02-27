package app

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

type commandRunner struct {
	command   docker.Command
	container *docker.Container
}
type containersScreenEventHandler struct {
	baseEventHandler
	widget appui.AppWidget
}

func (h *containersScreenEventHandler) handle(event *tcell.EventKey, f func(eventHandler)) {
	handled := h.handleKey(event.Key(), f)

	if !handled {
		handled = h.handleCharacter(event.Rune(), f)
	}
	if !handled {
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
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()

		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			f(h)
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
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()

		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			f(h)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {

				return
			}

			if err := dry.dockerDaemon.RestartContainer(id); err != nil {
				dry.message(
					fmt.Sprintf("Error restarting container %s, err: %s", id, err.Error()))
			}

		}()

	case docker.STOP:
		prompt := appui.NewPrompt(
			fmt.Sprintf("Do you want to stop container %s? (y/N)", id))
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()

		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			f(h)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {

				return
			}

			if err := dry.dockerDaemon.StopContainer(id); err != nil {
				dry.message(
					fmt.Sprintf("Error stopping container %s, err: %s", id, err.Error()))
			}

		}()

	case docker.LOGS:
		h.showLogs(id, false, f)
	case docker.RM:
		prompt := appui.NewPrompt(
			fmt.Sprintf("Do you want to remove container %s? (y/N)", id))
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()

		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			f(h)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {

				return
			}

			err := dry.dockerDaemon.Rm(id)
			if err == nil {
				dry.actionMessage(id, "removed")
			} else {
				dry.errorMessage(id, "removing", err)
			}
			refreshScreen()
		}()

	case docker.STATS:
		c := dry.dockerDaemon.ContainerByID(id)
		if c == nil || !docker.IsContainerRunning(c) {
			dry.message(
				fmt.Sprintf("Container with id %s not found or not running", id))
		} else {
			if statsChan, err := dry.dockerDaemon.StatsChannel(c); err != nil {
				dry.message(
					fmt.Sprintf("Error showing container stats: %s", err.Error()))
			} else {
				forwarder := newEventForwarder()
				f(forwarder)
				h.dry.changeView(NoView)
				go statsScreen(command.container, statsChan, screen, forwarder.events(), func() {
					h.dry.changeView(Main)
					f(h)
					refreshScreen()
				})
			}
		}

	case docker.INSPECT:
		forwarder := newEventForwarder()
		f(forwarder)
		err := inspect(
			h.screen,
			forwarder.events(),
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.Inspect(id)
			},
			func() {
				h.dry.changeView(Main)
				f(h)
				refreshScreen()
			})(id)

		if err != nil {
			dry.message(
				fmt.Sprintf("Error inspecting container: %s", err.Error()))
			return
		}

	case docker.HISTORY:
		history, err := dry.dockerDaemon.History(command.container.ImageID)

		if err == nil {
			forwarder := newEventForwarder()
			f(forwarder)
			renderer := appui.NewDockerImageHistoryRenderer(history)

			go appui.Less(renderer.String(), screen, forwarder.events(), func() {
				h.dry.changeView(Main)
				f(h)
			})
		} else {
			dry.message(
				fmt.Sprintf("Error showing image history: %s", err.Error()))
		}
	}
}

func (h *containersScreenEventHandler) handleCharacter(key rune, f func(eventHandler)) bool {
	handled := true
	dry := h.dry
	switch key {

	case '%': //filter containers
		forwarder := newEventForwarder()
		f(forwarder)
		applyFilter := func(filter string, canceled bool) {
			if !canceled {
				h.widget.Filter(filter)
			}
			f(h)
		}
		showFilterInput(newEventSource(forwarder.events()), applyFilter)
		refreshScreen()

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
			h.dry.message("There was an error removing the container: " + err.Error())
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
			h.dry.message("There was an error inspecting the container: " + err.Error())
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

			h.dry.message("There was an error showing logs: " + err.Error())
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
			h.dry.message("There was an error showing stats: " + err.Error())
		}
	default:
		handled = false
	}
	return handled
}

func (h *containersScreenEventHandler) handleKey(key tcell.Key, f func(eventHandler)) bool {
	handled := true
	cursor := h.screen.Cursor()
	switch key {
	case tcell.KeyF1: //sort
		h.widget.Sort()
		refreshScreen()
	case tcell.KeyF2: //show all containers
		cursor.Reset()
		widgets.ContainerList.ToggleShowAllContainers()
		refreshScreen()
	case tcell.KeyF5: // refresh
		h.dry.message("Refreshing container list")
		h.dry.dockerDaemon.Refresh(func(e error) {
			if e == nil {
				h.widget.Unmount()
				refreshScreen()
			} else {
				h.dry.message("There was an error refreshing: " + e.Error())
			}
		})
	case tcell.KeyCtrlE: //remove all stopped
		prompt := appui.NewPrompt(
			"All stopped containers will be removed. Do you want to continue? (y/N) ")
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()

		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			f(h)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {
				return
			}
			go func() {
				h.dry.message("<red>Removing all stopped containers</>")
				if count, err := h.dry.dockerDaemon.RemoveAllStoppedContainers(); err == nil {
					h.dry.message(fmt.Sprintf("<red>Removed %d stopped containers</>", count))
				} else {
					h.dry.message(
						fmt.Sprintf(
							"<red>Error removing all stopped containers: %s</>", err.Error()))
				}
				refreshScreen()
			}()
		}()

	case tcell.KeyCtrlK: //kill
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
			h.dry.message("There was an error killing container: " + err.Error())
		}
	case tcell.KeyCtrlL: //Logs with timestamp
		if err := h.widget.OnEvent(
			func(id string) error {
				container := h.dry.dockerDaemon.ContainerByID(id)
				if container == nil {
					return fmt.Errorf("Container with id %s not found", id)
				}
				h.showLogs(id, true, f)
				return nil
			}); err != nil {
			h.dry.message("There was an error showing logs: " + err.Error())
		}
	case tcell.KeyCtrlR: //start
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
			h.dry.message("There was an error restarting: " + err.Error())
		}
	case tcell.KeyCtrlT: //stop
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
			h.dry.message("There was an error stopping container: " + err.Error())
		}
	case tcell.KeyEnter: //Container menu
		showMenu := func(id string) error {
			h.screen.Cursor().Reset()
			widgets.ContainerMenu.ForContainer(id)
			widgets.ContainerMenu.OnUnmount = func() error {
				h.screen.Cursor().Reset()
				h.dry.changeView(Main)
				f(viewsToHandlers[Main])
				return refreshScreen()
			}
			h.dry.changeView(ContainerMenu)
			f(viewsToHandlers[ContainerMenu])
			return refreshScreen()
		}
		if err := h.widget.OnEvent(showMenu); err != nil {
			h.dry.message(err.Error())
		}

	default:
		handled = false
	}

	return handled
}

// statsScreen shows container stats on the screen
// TODO move to appui
func statsScreen(container *docker.Container, stats *docker.StatsChannel, screen *ui.Screen, events <-chan *tcell.EventKey, closeCallback func()) {
	defer closeCallback()

	if !docker.IsContainerRunning(container) {
		return
	}
	screen.ClearAndFlush()

	info, infoLines := appui.NewContainerInfo(container)
	screen.Render(1, info)
	d := ui.ActiveScreen.Dimensions()

	w, h := d.Width, d.Height
	header := appui.NewMonitorTableHeader()
	header.SetX(0)
	header.SetWidth(w)
	header.SetY(infoLines + 3)

	statsRow := appui.NewContainerStatsRow(container, header)
	statsRow.SetX(0)
	statsRow.SetY(header.Y + header.Height + 1)
	statsRow.SetWidth(w)

	t := time.NewTicker(1 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	sChan := stats.Start(ctx)
loop:
	for {
		select {

		case event := <-events:
			if event.Key() == tcell.KeyEsc {
				break loop
			}
		case stat, ok := <-sChan:
			{
				if !ok {
					break loop
				}
				statsRow.Update(stat)
				top, _ := appui.NewDockerTop(
					stat.ProcessList,
					0, statsRow.Y+statsRow.Height+2,
					h-infoLines-statsRow.GetHeight(),
					w)
				screen.RenderBufferer(
					header,
					top,
					statsRow)
				screen.Flush()
			}
		}
	}
	t.Stop()
	cancel()
	screen.Clear()
	screen.Sync()
}

func (h *containersScreenEventHandler) showLogs(id string, withTimestamp bool, f func(eventHandler)) {
	prompt := logsPrompt()
	widgets.add(prompt)
	forwarder := newEventForwarder()
	f(forwarder)
	refreshScreen()

	go func() {
		events := ui.EventSource{
			Events: forwarder.events(),
			EventHandledCallback: func(e *tcell.EventKey) error {
				return refreshScreen()
			},
		}
		prompt.OnFocus(events)
		widgets.remove(prompt)
		since, canceled := prompt.Text()

		if canceled {
			f(h)
			return
		}
		since = curateLogsDuration(since)
		logs, err := h.dry.dockerDaemon.Logs(id, since, withTimestamp)
		if err == nil {
			appui.Stream(logs, forwarder.events(), func() {
				h.dry.changeView(Main)
				f(h)
				refreshScreen()
			})
		} else {
			f(h)
			h.dry.message("Error showing container logs: " + err.Error())
		}
	}()
}
