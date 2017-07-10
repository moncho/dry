package app

import "github.com/nsf/termbox-go"

type nodesScreenEventHandler struct {
	baseEventHandler
}

func (h *nodesScreenEventHandler) handle(event termbox.Event) {
	handled := false

	switch event.Key {
	case termbox.KeyEnter:
		showServices := func(nodeID string) error {
			h.dry.ShowTasks(nodeID)
			h.renderChan <- struct{}{}
			return nil
		}
		if err := h.dry.state.activeWidget.OnEvent(showServices); err == nil {
			handled = true
		}
	}
	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(true)
	}
}

type taskScreenEventHandler struct {
	baseEventHandler
}

func (h *taskScreenEventHandler) handle(event termbox.Event) {

	switch event.Key {
	case termbox.KeyEsc:
		h.dry.ShowNodes()
	}

	h.baseEventHandler.handle(event)

}
