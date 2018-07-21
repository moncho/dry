package app

import (
	"github.com/moncho/dry/appui"
	termbox "github.com/nsf/termbox-go"
)

type monitorScreenEventHandler struct {
	baseEventHandler
	widget *appui.Monitor
}

func (h *monitorScreenEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	handled := false
	cursor := h.screen.Cursor
	switch event.Key {
	case termbox.KeyArrowUp: //cursor up
		handled = true
		cursor.ScrollCursorUp()
		h.widget.OnEvent(nil)
	case termbox.KeyArrowDown: // cursor down
		handled = true
		cursor.ScrollCursorDown()
		h.widget.OnEvent(nil)
	}
	if !handled {
		switch event.Ch {
		case 'g': //Cursor to the top
			handled = true
			cursor.Reset()
			h.widget.OnEvent(nil)

		case 'G': //Cursor to the bottom
			handled = true
			cursor.Bottom()
			h.widget.OnEvent(nil)

		case 'H', 'h', 'q', '1', '2', '3', '4', '5':
			handled = false
			cancelMonitorWidget()
		default:
			handled = true
		}
	}
	if !handled {
		h.baseEventHandler.handle(event, f)
	}
}
