package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type networksScreenEventHandler struct {
	baseEventbandler
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
	case termbox.KeyEnter: //inspect
		handled = true
		dry.InspectNetworkAt(cursorPos)
		focus = false
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.closeViewChan)
	case termbox.KeyCtrlE: //remove network
		handled = true
		if cursorPos >= 0 {
			network, err := dry.NetworkAt(cursorPos)
			if err == nil {
				dry.RemoveNetwork(network.ID)
				cursor.ScrollCursorDown()
			} else {
				ui.ShowErrorMessage(screen, h.keyboardQueueForView, h.closeViewChan, err)
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
			h.renderChan <- struct{}{}
		}
	} else {
		h.baseEventbandler.handle(event)
	}
}
