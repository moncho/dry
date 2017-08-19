package app

import (
	"github.com/moncho/dry/appui"
	termbox "github.com/nsf/termbox-go"
)

type servicesScreenEventHandler struct {
	baseEventHandler
}

func (h *servicesScreenEventHandler) handle(event termbox.Event) {
	handled := false
	focus := true

	switch event.Key {
	case termbox.KeyEnter:
		showServices := func(serviceID string) error {
			h.dry.ShowServiceTasks(serviceID)
			refreshScreen()
			return nil
		}
		h.dry.state.activeWidget.OnEvent(showServices)
		handled = true
	}
	switch event.Ch {
	case 'l':
		showServiceLogs := func(serviceID string) error {
			logs, err := h.dry.ServiceLogs(serviceID)
			if err == nil {
				go appui.Stream(h.screen, logs, h.eventChan, h.closeViewChan)
				return nil
			}
			return err
		}
		//TODO show error on screen
		if err := h.dry.state.activeWidget.OnEvent(showServiceLogs); err == nil {
			handled = true
			focus = false
		}
	}
	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(focus)
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
