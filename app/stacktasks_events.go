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
	if h.forwardingEvents() {
		h.eventChan <- event
		return
	}
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
		h.setForwardEvents(true)
		if err := h.widget.OnEvent(
			inspect(
				h.screen,
				h.eventChan,
				func(id string) (interface{}, error) {
					return h.dry.dockerDaemon.Task(id)
				},
				func() {
					h.dry.SetViewMode(StackTasks)
					h.setForwardEvents(false)
					f(h)
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
		h.setForwardEvents(true)
		applyFilter := func(filter string, canceled bool) {
			if !canceled {
				h.widget.Filter(filter)
			}
			h.setForwardEvents(false)
		}
		showFilterInput(newEventSource(h.eventChan), applyFilter)
	}

	if !handled {
		h.baseEventHandler.handle(event, f)
	} else {
		refreshScreen()
	}

}
