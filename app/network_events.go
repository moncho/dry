package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type networksScreenEventHandler struct {
	baseEventHandler
}

func (h *networksScreenEventHandler) widget() appui.EventableWidget {
	return h.dry.widgetRegistry.Networks
}

func (h *networksScreenEventHandler) handle(event termbox.Event) {
	focus := true
	dry := h.dry
	screen := h.screen
	cursor := screen.Cursor
	cursorPos := cursor.Position()
	handled := false
	switch event.Key {
	case termbox.KeyF1: //sort
		handled = true
		dry.SortNetworks()
	case termbox.KeyF5: // refresh
		handled = true
		h.widget().Unmount()
	case termbox.KeyEnter: //inspect
		handled = true
		dry.InspectNetworkAt(cursorPos)
		focus = false
		go appui.Less(renderDry(dry), screen, h.eventChan, h.closeViewChan)
	case termbox.KeyCtrlE: //remove network
		handled = true
		if cursorPos >= 0 {
			network, err := dry.NetworkAt(cursorPos)
			if err == nil {
				dry.RemoveNetwork(network.ID)
			} else {
				ui.ShowErrorMessage(screen, h.eventChan, h.closeViewChan, err)
			}
		}
	}
	if !handled {
		switch event.Ch {
		case '3':
			//already in network screen
			handled = true

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
