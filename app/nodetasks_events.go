package app

import (
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/appui/swarm"
)

type taskScreenEventHandler struct {
	baseEventHandler
	widget *swarm.NodeTasksWidget
}

func (h *taskScreenEventHandler) handle(event *tcell.EventKey, f func(eventHandler)) {

	handled := true
	switch event.Key() {
	case tcell.KeyEsc:
		f(viewsToHandlers[Nodes])
		h.dry.changeView(Nodes)
	case tcell.KeyF1: //sort
		widgets.NodeTasks.Sort()
	case tcell.KeyF5: // refresh
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
					h.dry.changeView(Tasks)
					f(h)
					refreshScreen()
				})); err != nil {
			h.dry.message(
				fmt.Sprintf("Error inspecting stack: %s", err.Error()))
		}
	default:
		handled = false
	}
	if !handled {
		switch event.Rune() {
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
