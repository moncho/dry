package app

import (
	"fmt"
	"strings"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/json"
	termbox "github.com/nsf/termbox-go"
)

type cMenuEventHandler struct {
	baseEventHandler
}

func (h *cMenuEventHandler) widget() appui.AppWidget {
	return h.dry.widgetRegistry.ContainerMenu
}

func (h *cMenuEventHandler) handle(event termbox.Event) {
	if h.forwardingEvents {
		h.eventChan <- event
		return
	}
	handled := false

	switch event.Key {

	case termbox.KeyEsc:
		handled = true
		h.screen.Cursor.Reset()
		h.dry.ShowContainers()
	case termbox.KeyEnter:
		handled = true
		err := h.widget().OnEvent(func(s string) error {
			//s is a string made of two parts: an Id and a description
			//separated by ":"
			cd := strings.Split(s, ":")
			id := cd[0]
			command, err := docker.CommandFromDescription(cd[1])
			if err != nil {
				return err
			}
			h.handleCommand(id, command)
			return nil
		})
		if err != nil {
			h.dry.appmessage(fmt.Sprintf("Could not run command: %s", err.Error()))
		}
	}

	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		refreshScreen()
	}
}

func (h *cMenuEventHandler) handleCommand(id string, command docker.Command) {

	dry := h.dry
	screen := h.screen

	container := dry.dockerDaemon.ContainerByID(id)
	switch command {
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
					h.dry.SetViewMode(ContainerMenu)
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

		h.setForwardEvents(true)
		statsChan := dry.dockerDaemon.OpenChannel(container)
		go statsScreen(container, statsChan, screen, h.eventChan,
			func() {
				h.setForwardEvents(false)
				h.dry.SetViewMode(ContainerMenu)
				h.closeViewChan <- struct{}{}
			})

	case docker.INSPECT:
		h.setFocus(false)
		container, err := h.dry.dockerDaemon.Inspect(id)
		if err == nil {
			go func() {
				defer func() {
					h.setFocus(true)
					h.dry.SetViewMode(ContainerMenu)
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
		}
	case docker.HISTORY:
		history, err := dry.dockerDaemon.History(container.ImageID)

		if err == nil {
			renderer := appui.NewDockerImageHistoryRenderer(history)
			h.setFocus(false)
			go appui.Less(renderer, screen, h.eventChan, h.closeViewChan)
		} else {
			dry.appmessage(
				fmt.Sprintf("Error showing image history: %s", err.Error()))
		}
	}
}
