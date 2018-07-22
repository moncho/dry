package app

import (
	"fmt"

	"github.com/moncho/dry/appui"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

type networksScreenEventHandler struct {
	baseEventHandler
	widget *appui.DockerNetworksWidget
}

func (h *networksScreenEventHandler) handle(event termbox.Event, f func(eh eventHandler)) {
	dry := h.dry
	screen := h.screen
	handled := true
	switch event.Key {
	case termbox.KeyF1: //sort
		h.widget.Sort()
		refreshScreen()
	case termbox.KeyF5: // refresh
		h.dry.appmessage("Refreshing network list")
		h.widget.Unmount()
		refreshScreen()
	case termbox.KeyEnter: //inspect
		forwarder := newEventForwarder()
		f(forwarder)
		inspectNetwork := inspect(screen, forwarder.events(),
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.NetworkInspect(id)
			},
			func() {
				h.dry.SetViewMode(Networks)
				f(h)
				refreshScreen()
			})

		if err := h.widget.OnEvent(inspectNetwork); err != nil {
			dry.appmessage(
				fmt.Sprintf("Error inspecting image: %s", err.Error()))
		}

	case termbox.KeyCtrlE: //remove network

		prompt := appui.NewPrompt("Do you want to remove the selected network? (y/N)")
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e termbox.Event) error {
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
					h.dry.appmessage(fmt.Sprintf("<red>Removed network:</> <white>%s</>", shortID))
				} else {
					h.dry.appmessage(fmt.Sprintf("<red>Error network image </><white>%s: %s</>", shortID, err.Error()))
				}

				return nil
			}
			if err := h.widget.OnEvent(rmNetwork); err != nil {
				dry.appmessage(
					fmt.Sprintf("Error removing network: %s", err.Error()))
			}
			refreshScreen()

		}()

	default:
		handled = false
	}
	if !handled {
		switch event.Ch {
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
