package app

import (
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/ui"
)

type stacksScreenEventHandler struct {
	baseEventHandler
	widget *swarm.StacksWidget
}

func (h *stacksScreenEventHandler) handle(event *tcell.EventKey, f func(eventHandler)) {
	handled := true
	switch event.Key() {
	case tcell.KeyF1: //sort
		widgets.Stacks.Sort()
	case tcell.KeyF5: // refresh
		h.dry.message("Refreshing stack list")
		h.widget.Unmount()
	case tcell.KeyEnter: //inspect
		showTasks := func(stack string) error {
			widgets.StackTasks.ForStack(stack)
			h.dry.changeView(StackTasks)
			f(viewsToHandlers[StackTasks])
			return refreshScreen()
		}
		h.widget.OnEvent(showTasks)
	case tcell.KeyCtrlR: //remove stack
		rw := appui.NewPrompt("The selected stack will be removed. Do you want to proceed? y/N")
		widgets.add(rw)
		forwarder := newRegisteredEventForwarder(f)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
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
			removeStack := func(stack string) error {
				err := h.dry.dockerDaemon.StackRemove(stack)
				if err == nil {
					h.dry.message(fmt.Sprintf("Stack %s removed", stack))
				}
				return err
			}
			if err := h.widget.OnEvent(removeStack); err != nil {
				h.dry.message("There was an error removing the stack: " + err.Error())
			}
			refreshScreen()
		}()
	default:
		handled = false
	}
	if !handled {
		switch event.Rune() {
		case '7':
			//already in stack screen
			handled = true
		case '%':
			handled = true
			forwarder := newRegisteredEventForwarder(f)
			applyFilter := func(filter string, canceled bool) {
				if !canceled {
					h.widget.Filter(filter)
				}
				f(h)
			}
			showFilterInput(newEventSource(forwarder.events()), applyFilter)
		}
	}
	if handled {
		refreshScreen()
	} else {
		h.baseEventHandler.handle(event, f)
	}
}
