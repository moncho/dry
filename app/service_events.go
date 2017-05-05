package app

import termbox "github.com/nsf/termbox-go"

type servicesScreenEventHandler struct {
	baseEventHandler
}

func (h *servicesScreenEventHandler) handle(event termbox.Event) {
	handled := false

	switch event.Key {
	case termbox.KeyEnter:
		showServices := func(serviceID string) error {
			h.dry.ShowServiceTasks(serviceID)
			h.renderChan <- struct{}{}
			return nil
		}
		h.dry.state.activeWidget.OnEvent(showServices)
		handled = true
	}
	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(true)
	}
}

type serviceTaskScreenEventHandler struct {
	baseEventHandler
}

func (h *serviceTaskScreenEventHandler) handle(event termbox.Event) {

	switch event.Key {
	case termbox.KeyEsc:
		h.dry.ShowServices()
	}

	h.baseEventHandler.handle(event)

}
