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
	handled := true
	dry := h.dry

	switch event.Key {
	case termbox.KeyF1: // sort
		widgets.ServiceList.Sort()
	case termbox.KeyF5: // refresh
		h.dry.appmessage("Refreshing the service list")
		if err := h.widget.Unmount(); err != nil {
			h.dry.appmessage("There was an error refreshing the service list: " + err.Error())
		}
	case termbox.KeyCtrlR:
		rw := appui.NewPrompt("The selected service will be removed. Do you want to proceed? y/N")
		widgets.add(rw)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			rw.OnFocus(events)
			widgets.remove(rw)
			confirmation, canceled := rw.Text()
			f(h)
			if canceled || (confirmation != "y" && confirmation != "Y") {
				return
			}
			removeService := func(serviceID string) error {
				err := dry.dockerDaemon.ServiceRemove(serviceID)
				return err
			}
			if err := h.widget.OnEvent(removeService); err != nil {
				h.dry.appmessage("There was an error removing the service: " + err.Error())
			}
			refreshScreen()
		}()

	case termbox.KeyCtrlS:

		rw := appui.NewPrompt("Scale service. Number of replicas?")
		widgets.add(rw)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			rw.OnFocus(events)
			widgets.remove(rw)
			replicas, canceled := rw.Text()
			f(h)
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
				return err
			}
			if err := h.widget.OnEvent(scaleService); err != nil {
				h.dry.appmessage("There was an error scaling the service: " + err.Error())
			}
			refreshScreen()
		}()
	case termbox.KeyCtrlU: //Update service
		rw := appui.NewPrompt("The selected service will be updated. Do you want to proceed? y/N")
		widgets.add(rw)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			rw.OnFocus(events)
			widgets.remove(rw)
			confirmation, canceled := rw.Text()
			f(h)
			if canceled || (confirmation != "y" && confirmation != "Y") {
				return
			}
			removeService := func(serviceID string) error {
				err := dry.dockerDaemon.ServiceUpdate(serviceID)
				return err
			}
			if err := h.widget.OnEvent(removeService); err != nil {
				h.dry.appmessage("There was an error updating the service: " + err.Error())
			}
			refreshScreen()
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
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		applyFilter := func(filter string, canceled bool) {
			if !canceled {
				h.widget.Filter(filter)
			}
			f(h)
		}
		showFilterInput(newEventSource(forwarder.events()), applyFilter)
	case 'i' | 'I':
		handled = true
		forwarder := newEventForwarder()
		f(forwarder)
		inspectService := inspect(
			h.screen,
			forwarder.events(),
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.Service(id)
			},
			func() {
				h.dry.SetViewMode(Services)
				f(h)
				refreshScreen()
			})

		if err := h.widget.OnEvent(inspectService); err != nil {
			h.dry.appmessage("There was an error inspecting the service: " + err.Error())
		}

	case 'l':
		prompt := logsPrompt()
		handled = true
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			prompt.OnFocus(newEventSource(forwarder.events()))
			widgets.remove(prompt)
			since, canceled := prompt.Text()

			if canceled {
				f(h)
				return
			}

			showServiceLogs := func(serviceID string) error {
				logs, err := h.dry.dockerDaemon.ServiceLogs(serviceID, since)
				if err == nil {
					appui.Stream(logs, forwarder.events(),
						func() {
							h.dry.SetViewMode(Services)
							f(h)
							refreshScreen()
						})
					return nil
				}
				return err
			}
			if err := h.widget.OnEvent(showServiceLogs); err != nil {
				h.dry.appmessage("There was an error showing service logs: " + err.Error())
				f(h)
			}
		}()
	}
	if !handled {
		h.baseEventHandler.handle(event, f)
	}
}
