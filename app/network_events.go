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
	if h.forwardingEvents() {
		h.eventChan <- event
		return
	}
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
		inspectNetwork := inspect(screen, h.eventChan,
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.NetworkInspect(id)
			},
			func() {
				h.dry.SetViewMode(Images)
				f(viewsToHandlers[Images])
				h.setForwardEvents(false)
				refreshScreen()
			})
		h.setForwardEvents(true)
		if err := h.widget.OnEvent(inspectNetwork); err != nil {
			dry.appmessage(
				fmt.Sprintf("Error inspecting image: %s", err.Error()))
		}

	case termbox.KeyCtrlE: //remove network

		prompt := appui.NewPrompt("Do you want to remove the selected network? (y/N)")
		widgets.add(prompt)
		h.setForwardEvents(true)
		refreshScreen()
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
			h.setForwardEvents(true)
			applyFilter := func(filter string, canceled bool) {
				if !canceled {
					h.widget.Filter(filter)
				}
				h.setForwardEvents(false)
			}
			showFilterInput(newEventSource(h.eventChan), applyFilter)
		}
	}
	if !handled {
		h.baseEventHandler.handle(event, f)
	}
}
