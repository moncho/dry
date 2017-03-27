package app

import "github.com/nsf/termbox-go"

type monitorScreenEventHandler struct {
	baseEventHandler
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
	switch event.Ch {
	case 'g': //Cursor to the top
		cursor.Reset()
		handled = true
	case 'G': //Cursor to the bottom
		cursor.Bottom()
		handled = true
	case 'H', 'h', 'q', '1', '2', '3':
		handled = false
	default:
		handled = true
	}
	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(true)
	}
}
