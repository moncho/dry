package app

import termbox "github.com/nsf/termbox-go"

type dfScreenEventHandler struct {
	baseEventHandler
}

func (h *dfScreenEventHandler) handle(event termbox.Event) {
	h.setFocus(true)
}
