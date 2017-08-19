package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

type servicesScreenEventHandler struct {
	baseEventHandler
	passingEvents bool
}

func (h *servicesScreenEventHandler) handle(event termbox.Event) {
	if h.passingEvents {
		h.eventChan <- event
		return
	}
	handled := false
	focus := true
	dry := h.dry
	switch event.Key {
	case termbox.KeyCtrlR:

		rw := appui.NewAskForConfirmation("About to remove the selected service. Do you want to proceed? y/N")
		h.passingEvents = true
		handled = true
		dry.widgetRegistry.add(rw)
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			rw.OnFocus(events)
			dry.widgetRegistry.remove(rw)
			confirmation, canceled := rw.Text()
			h.passingEvents = false
			if canceled || (confirmation != "y" && confirmation != "Y") {
				return
			}
			removeService := func(serviceID string) error {
				err := dry.dockerDaemon.ServiceRemove(serviceID)
				refreshScreen()
				return err
			}
			h.dry.state.activeWidget.OnEvent(removeService)
		}()

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
		if h.hasFocus() {
			refreshScreen()
		}
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
