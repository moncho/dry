package app

import (
	"fmt"

	"github.com/moncho/dry/appui/swarm"
	termbox "github.com/nsf/termbox-go"
)

type stackTasksScreenEventHandler struct {
	baseEventHandler
	widget *swarm.StacksTasksWidget
}

func (h *stackTasksScreenEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	handled := true

	switch event.Key {
	case termbox.KeyEsc:
		f(viewsToHandlers[Stacks])
		h.dry.SetViewMode(Stacks)
	case termbox.KeyF1: //sort
		h.widget.Sort()
	case termbox.KeyF5: // refresh
		h.dry.appmessage("Refreshing stack tasks list")
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
					h.dry.SetViewMode(StackTasks)
					f(h)
					refreshScreen()
				})); err != nil {
			h.dry.appmessage(
				fmt.Sprintf("Error inspecting stack: %s", err.Error()))
		}
	default:
		handled = false
	}
	switch event.Ch {
	case '%':
		handled = true
		forwarder := newEventForwarder()
		f(forwarder)
		applyFilter := func(filter string, canceled bool) {
			if !canceled {
				h.widget.Filter(filter)
			}
			f(h)
		}
		showFilterInput(newEventSource(forwarder.events()), applyFilter)
	}

	if !handled {
		h.baseEventHandler.handle(event, f)
	}
}
