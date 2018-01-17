package app

import (
	"fmt"

	"github.com/moncho/dry/appui"
	termbox "github.com/nsf/termbox-go"
)

type networksScreenEventHandler struct {
	baseEventHandler
}

func (h *networksScreenEventHandler) widget() appui.AppWidget {
	return h.dry.widgetRegistry.Networks
}

func (h *networksScreenEventHandler) handle(event termbox.Event) {
	if h.forwardingEvents {
		h.eventChan <- event
		return
	}
	focus := true
	dry := h.dry
	screen := h.screen
	handled := false
	switch event.Key {
	case termbox.KeyF1: //sort
		handled = true
		h.widget().Sort()
	case termbox.KeyF5: // refresh
		handled = true
		h.dry.appmessage("Refreshing network list")
		h.widget().Unmount()

	case termbox.KeyEnter: //inspect
		handled = true
		inspectNetwork := func(id string) error {
			network, err := h.dry.dockerDaemon.NetworkInspect(id)
			if err != nil {
				return err
			}
			focus = false
			renderer := appui.NewJSONRenderer(network)
			go appui.Less(renderer, screen, h.eventChan, h.closeViewChan)
			return nil
		}
		if err := h.widget().OnEvent(inspectNetwork); err != nil {
			dry.appmessage(
				fmt.Sprintf("Error inspecting image: %s", err.Error()))
		}

	case termbox.KeyCtrlE: //remove network
		handled = true
		rmNetwork := func(id string) error {
			dry.RemoveNetwork(id)
			return nil
		}
		if err := h.widget().OnEvent(rmNetwork); err != nil {
			dry.appmessage(
				fmt.Sprintf("Error removing network: %s", err.Error()))
		}

	}
	if !handled {
		switch event.Ch {
		case '3':
			//already in network screen
			handled = true
		case '%':
			handled = true
			showFilterInput(h)
		}
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
