package app

import (
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/appui"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

type networksScreenEventHandler struct {
	baseEventHandler
	widget *appui.DockerNetworksWidget
}

func (h *networksScreenEventHandler) handle(event *tcell.EventKey, f func(eh eventHandler)) {
	dry := h.dry
	screen := h.screen
	handled := true
	switch event.Key() {
	case tcell.KeyF1: //sort
		h.widget.Sort()
		refreshScreen()
	case tcell.KeyF5: // refresh
		h.dry.message("Refreshing network list")
		h.widget.Unmount()
		refreshScreen()
	case tcell.KeyEnter: //inspect
		forwarder := newEventForwarder()
		f(forwarder)
		inspectNetwork := inspect(screen, forwarder.events(),
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.NetworkInspect(id)
			},
			func() {
				h.dry.changeView(Networks)
				f(h)
				refreshScreen()
			})

		if err := h.widget.OnEvent(inspectNetwork); err != nil {
			dry.message(
				fmt.Sprintf("Error inspecting network: %s", err.Error()))
		}

	case tcell.KeyCtrlE: //remove network

		prompt := appui.NewPrompt("Do you want to remove the selected network? (y/N)")
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

			rmNetwork := func(id string) error {
				shortID := drydocker.TruncateID(id)
				if err := h.dry.dockerDaemon.RemoveNetwork(id); err == nil {
					h.dry.message(fmt.Sprintf("<red>Removed network:</> <white>%s</>", shortID))
				} else {
					h.dry.message(fmt.Sprintf("<red>Error removing network </><white>%s: %s</>", shortID, err.Error()))
				}

				return nil
			}
			if err := h.widget.OnEvent(rmNetwork); err != nil {
				dry.message(
					fmt.Sprintf("Error removing network: %s", err.Error()))
			}
			refreshScreen()

		}()

	default:
		handled = false
	}
	if !handled {
		switch event.Rune() {
		case '3':
			//already in network screen
			handled = true
		case '%':
			handled = true
			forwarder := newEventForwarder()
			f(forwarder)
			refreshScreen()
			applyFilter := func(filter string, canceled bool) {
				if !canceled {
					h.widget.Filter(filter)
				}
				f(h)
			}
			showFilterInput(newEventSource(forwarder.events()), applyFilter)
		}
	}
	if !handled {
		h.baseEventHandler.handle(event, f)
	}
}
