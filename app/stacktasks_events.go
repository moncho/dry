package app

import (
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/appui/swarm"
)

type stackTasksScreenEventHandler struct {
	baseEventHandler
	widget *swarm.StacksTasksWidget
}

func (h *stackTasksScreenEventHandler) handle(event *tcell.EventKey, f func(eventHandler)) {
	handled := true

	switch event.Key() {
	case tcell.KeyEsc:
		f(viewsToHandlers[Stacks])
		h.dry.changeView(Stacks)
	case tcell.KeyF1: //sort
		h.widget.Sort()
	case tcell.KeyF5: // refresh
		h.dry.message("Refreshing stack tasks list")
		h.widget.Unmount()
	case tcell.KeyEnter:
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
					h.dry.changeView(StackTasks)
					f(h)
					refreshScreen()
				})); err != nil {
			h.dry.message(
				fmt.Sprintf("Error inspecting stack: %s", err.Error()))
		}
	default:
		handled = false
	}
	switch event.Rune() {
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

	if handled {
		refreshScreen()
	} else {
		h.baseEventHandler.handle(event, f)
	}
}
