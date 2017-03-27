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
	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(true)
	}
}
