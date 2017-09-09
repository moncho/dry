package app

import (
	"fmt"
	"strconv"

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

	case termbox.KeyCtrlS:

		rw := appui.NewAskForConfirmation("Scale service. Number of replicas?")
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
			replicas, canceled := rw.Text()
			h.passingEvents = false
			if canceled {
				return
			}
			scaleTo, err := strconv.Atoi(replicas)
			if err != nil || scaleTo < 0 {
				dry.appmessage(
					fmt.Sprintf("Cannot scale service, invalid number of replicas: %s", replicas))
				return
			}

			scaleService := func(serviceID string) error {
				err := dry.dockerDaemon.ServiceScale(serviceID, uint64(scaleTo))

				if err == nil {
					dry.appmessage(fmt.Sprintf("Service %s scaled to %d replicas", serviceID, scaleTo))
				}
				refreshScreen()
				return err
			}
			h.dry.state.activeWidget.OnEvent(scaleService)
		}()

	case termbox.KeyEnter:
		showServices := func(serviceID string) error {
			h.dry.ShowServiceTasks(serviceID)
			return refreshScreen()
		}
		h.dry.state.activeWidget.OnEvent(showServices)
		handled = true
	}
	switch event.Ch {
	case 'i' | 'I':
		handled = true

		inspectService := func(serviceID string) error {
			service, err := h.dry.ServiceInspect(serviceID)
			if err == nil {
				go appui.Less(
					appui.NewJSONRenderer(service),
					h.screen, h.eventChan, h.closeViewChan)
				return nil
			}
			return err
		}
		if err := h.dry.widgetRegistry.ServiceList.OnEvent(inspectService); err == nil {
			focus = false
		}

	case 'l':
		handled = true

		showServiceLogs := func(serviceID string) error {
			logs, err := h.dry.ServiceLogs(serviceID)
			if err == nil {
				go appui.Stream(h.screen, logs, h.eventChan, h.closeViewChan)
				return nil
			}
			return err
		}
		//TODO show error on screen
		if h.dry.state.activeWidget != nil {
			if err := h.dry.state.activeWidget.OnEvent(showServiceLogs); err == nil {
				focus = false
			}
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
