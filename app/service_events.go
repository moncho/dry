package app

import (
	"fmt"
	"strconv"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

type servicesScreenEventHandler struct {
	baseEventHandler
	widget *swarm.ServicesWidget
}

func (h *servicesScreenEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	if h.forwardingEvents() {
		h.eventChan <- event
		return
	}
	handled := true
	dry := h.dry

	switch event.Key {
	case termbox.KeyF1: // refresh
		widgets.ServiceList.Sort()
	case termbox.KeyF5: // refresh
		h.dry.appmessage("Refreshing the service list")
		if err := h.widget.Unmount(); err != nil {
			h.dry.appmessage("There was an error refreshing the service list: " + err.Error())
		}
	case termbox.KeyCtrlR:

		rw := appui.NewPrompt("The selected service will be removed. Do you want to proceed? y/N")
		h.setForwardEvents(true)
		widgets.add(rw)
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			rw.OnFocus(events)
			widgets.remove(rw)
			confirmation, canceled := rw.Text()
			h.setForwardEvents(false)
			if canceled || (confirmation != "y" && confirmation != "Y") {
				return
			}
			removeService := func(serviceID string) error {
				err := dry.dockerDaemon.ServiceRemove(serviceID)
				refreshScreen()
				return err
			}
			if err := h.widget.OnEvent(removeService); err != nil {
				h.dry.appmessage("There was an error removing the service: " + err.Error())
			}
		}()

	case termbox.KeyCtrlS:

		rw := appui.NewPrompt("Scale service. Number of replicas?")
		h.setForwardEvents(true)
		widgets.add(rw)
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			rw.OnFocus(events)
			widgets.remove(rw)
			replicas, canceled := rw.Text()
			h.setForwardEvents(false)
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
			if err := h.widget.OnEvent(scaleService); err != nil {
				h.dry.appmessage("There was an error scaling the service: " + err.Error())
			}
		}()

	case termbox.KeyEnter:
		showTasks := func(serviceID string) error {
			h.screen.Cursor.Reset()
			widgets.ServiceTasks.ForService(serviceID)
			f(viewsToHandlers[ServiceTasks])
			dry.SetViewMode(ServiceTasks)
			return refreshScreen()
		}
		h.widget.OnEvent(showTasks)
	default:
		handled = false
	}
	switch event.Ch {
	case '%':
		handled = true
		h.setForwardEvents(true)
		applyFilter := func(filter string, canceled bool) {
			if !canceled {
				h.widget.Filter(filter)
			}
			h.setForwardEvents(false)
		}
		showFilterInput(newEventSource(h.eventChan), applyFilter)
	case 'i' | 'I':
		handled = true
		h.setForwardEvents(true)
		inspectService := inspect(
			h.screen,
			h.eventChan,
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.Service(id)
			},
			func() {
				h.setForwardEvents(false)
				h.dry.SetViewMode(Services)
				f(h)
				refreshScreen()
			})

		if err := h.widget.OnEvent(inspectService); err != nil {
			h.dry.appmessage("There was an error inspecting the service: " + err.Error())
		}

	case 'l':

		prompt := logsPrompt()
		h.setForwardEvents(true)
		handled = true
		widgets.add(prompt)
		go func() {
			prompt.OnFocus(newEventSource(h.eventChan))
			widgets.remove(prompt)
			since, canceled := prompt.Text()

			if canceled {
				h.setForwardEvents(false)
				return
			}

			showServiceLogs := func(serviceID string) error {
				logs, err := h.dry.dockerDaemon.ServiceLogs(serviceID, since)
				if err == nil {
					appui.Stream(logs, h.eventChan,
						func() {
							h.setForwardEvents(false)
							h.dry.SetViewMode(Services)
							f(h)
						})
					return nil
				}
				return err
			}
			if err := h.widget.OnEvent(showServiceLogs); err != nil {
				h.dry.appmessage("There was an error showing service logs: " + err.Error())
				h.setForwardEvents(false)

			}
		}()
	}
	if !handled {
		h.baseEventHandler.handle(event, f)
	} else {
		refreshScreen()
	}
}
