package app

import "github.com/nsf/termbox-go"

type nodesScreenEventHandler struct {
	baseEventHandler
}

func (h *nodesScreenEventHandler) handle(event termbox.Event) {
	handled := false
	switch event.Key {
	case termbox.KeyEnter:
		//h.dry.ShowSelectedNodeTasks()
		handled = true
	}
	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(true)
	}
}
