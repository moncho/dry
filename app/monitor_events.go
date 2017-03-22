package app

import "github.com/nsf/termbox-go"

type monitorScreenEventHandler struct {
	baseEventHandler
}

func (h *monitorScreenEventHandler) handle(event termbox.Event) {
	ignored := false

	switch event.Key {
	case termbox.KeyArrowUp:
		//To avoid the base handler handling this
		ignored = true
	case termbox.KeyArrowDown:
		//To avoid the base handler handling this
		ignored = true
	case termbox.KeyArrowLeft:
		//To avoid the base handler handling this
		ignored = true
	case termbox.KeyArrowRight:
		//To avoid the base handler handling this
		ignored = true
	}
	if !ignored {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(true)
	}
}
