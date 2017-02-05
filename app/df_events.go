package app

import termbox "github.com/nsf/termbox-go"

type dfScreenEventHandler struct {
	baseEventbandler
}

func (h *dfScreenEventHandler) handle(event termbox.Event) {
	h.setFocus(true)
}
