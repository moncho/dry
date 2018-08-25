package app

import (
	"fmt"

	"github.com/moncho/dry/appui/swarm"
	termbox "github.com/nsf/termbox-go"
)

type serviceTasksScreenEventHandler struct {
	baseEventHandler
	widget *swarm.ServiceTasksWidget
}

func (h *serviceTasksScreenEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	handled := true

	switch event.Key {
	case termbox.KeyEsc:
		f(viewsToHandlers[Services])
		h.dry.ViewMode(Services)
		refreshScreen()
	case termbox.KeyF1: //sort
		widgets.ServiceTasks.Sort()
	case termbox.KeyF5: // refresh
		h.widget.Unmount()
	case termbox.KeyEnter:
		forwarder := newEventForwarder()
		f(forwarder)
		if err := h.widget.OnEvent(
			inspect(
				h.screen,
				forwarder.events(),
				func(id string) (interface{}, error) {
					return h.dry.dockerDaemon.Task(id)
				},
				func() {
					h.dry.ViewMode(ServiceTasks)
					f(h)
					refreshScreen()
				})); err != nil {
			h.dry.appmessage(
				fmt.Sprintf("Error inspecting stack: %s", err.Error()))
		}

	default:
		handled = false
	}
	if !handled {
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
		}
	}
	if !handled {
		h.baseEventHandler.handle(event, f)
	}
}
