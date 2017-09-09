package app

import (
	"github.com/moncho/dry/appui"
	"github.com/nsf/termbox-go"
)

type monitorScreenEventHandler struct {
	baseEventHandler
}

func (h *monitorScreenEventHandler) widget() appui.EventableWidget {
	return h.dry.state.activeWidget
}

func (h *monitorScreenEventHandler) handle(event termbox.Event) {
	handled := false
	cursor := h.screen.Cursor
	switch event.Key {
	case termbox.KeyArrowUp: //cursor up
		cursor.ScrollCursorUp()
		handled = true
	case termbox.KeyArrowDown: // cursor down
		cursor.ScrollCursorDown()
		handled = true
	}
	if !handled {
		switch event.Ch {
		case 'g': //Cursor to the top
			cursor.Reset()
			handled = true
		case 'G': //Cursor to the bottom
			cursor.Bottom()
			handled = true
		case 'H', 'h', 'q', '1', '2', '3', '4', '5':
			handled = false
			cancelMonitorWidget()
		default:
			handled = true
		}
	}
	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(true)
	}
}
