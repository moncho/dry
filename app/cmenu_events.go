package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

type cMenuEventHandler struct {
	baseEventHandler
}

func (h *cMenuEventHandler) handle(event *tcell.EventKey, f func(eventHandler)) {

	handled := true
	switch event.Key() {

	case tcell.KeyEsc:
		widgets.ContainerMenu.Unmount()
		refreshScreen()

	case tcell.KeyEnter:
		err := widgets.ContainerMenu.OnEvent(func(s string) error {
			//s is a string made of two parts: an Id and a description
			//separated by ":"
			cd := strings.Split(s, ":")
			if len(cd) != 2 {
				return errors.New("Invalid command description: " + s)
			}
			id := cd[0]
			command, err := docker.CommandFromDescription(cd[1])
			if err != nil {
				return err
			}
			h.handleCommand(id, command, f)
			return nil
		})
		if err != nil {
			h.dry.message(fmt.Sprintf("Could not run command: %s", err.Error()))
		}
	default:
		handled = false
	}

	if !handled {
		h.baseEventHandler.handle(event, f)
	}
}

func (h *cMenuEventHandler) handleCommand(id string, command docker.Command, f func(eventHandler)) {
	dry := h.dry
	screen := h.screen

	container := dry.dockerDaemon.ContainerByID(id)
	switch command {
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
				widgets.ContainerMenu.ForContainer(id)
			} else {
				dry.errorMessage(id, "killing", err)
			}
			refreshScreen()
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

			if err := dry.dockerDaemon.RestartContainer(id); err == nil {
				widgets.ContainerMenu.ForContainer(id)
			} else {
				dry.message(
					fmt.Sprintf("Error restarting container %s, err: %s", id, err.Error()))
			}
			refreshScreen()
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

			dry.actionMessage(id, "Stopping")
			err := dry.dockerDaemon.StopContainer(id)
			if err == nil {
				widgets.ContainerMenu.ForContainer(id)
			} else {
				dry.errorMessage(id, "stopping", err)
			}
			refreshScreen()
		}()
	case docker.LOGS:
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

			logs, err := h.dry.dockerDaemon.Logs(id, since, false)
			if err == nil {
				appui.Stream(logs, forwarder.events(),
					func() {
						h.dry.changeView(ContainerMenu)
						f(h)
						refreshScreen()
					})
			} else {
				f(h)
				h.dry.message("Error showing container logs: " + err.Error())
			}
		}()
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

			dry.actionMessage(id, "Removing")
			err := dry.dockerDaemon.Rm(id)
			if err == nil {
				dry.actionMessage(id, "removed")
				widgets.ContainerMenu.Unmount()
			} else {
				dry.errorMessage(id, "removing", err)
			}
			refreshScreen()
		}()

	case docker.STATS:
		forwarder := newEventForwarder()
		f(forwarder)
		h.dry.changeView(NoView)
		if statsChan, err := dry.dockerDaemon.StatsChannel(container); err != nil {
			dry.message(
				fmt.Sprintf("Error showing container stats: %s", err.Error()))
		} else {
			go statsScreen(container, statsChan, screen, forwarder.events(),
				func() {
					h.dry.changeView(ContainerMenu)
					f(h)
					refreshScreen()
				})
		}

	case docker.INSPECT:
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()

		err := inspect(
			h.screen,
			forwarder.events(),
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.Inspect(id)
			},
			func() {
				h.dry.changeView(ContainerMenu)
				f(h)
				refreshScreen()
			})(id)

		if err != nil {
			dry.message(
				fmt.Sprintf("Error inspecting container: %s", err.Error()))
			return
		}
	case docker.HISTORY:
		history, err := dry.dockerDaemon.History(container.ImageID)

		if err == nil {
			renderer := appui.NewDockerImageHistoryRenderer(history)
			forwarder := newEventForwarder()
			f(forwarder)
			refreshScreen()
			go appui.Less(renderer.String(), screen, forwarder.events(), func() {
				h.dry.changeView(ContainerMenu)
				f(h)
				refreshScreen()
			})
		} else {
			dry.message(
				fmt.Sprintf("Error showing image history: %s", err.Error()))
		}
	}
}
