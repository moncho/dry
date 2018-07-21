package app

import (
	"fmt"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

type stacksScreenEventHandler struct {
	baseEventHandler
	widget *swarm.StacksWidget
}

func (h *stacksScreenEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	if h.forwardingEvents() {
		h.eventChan <- event
		return
	}
	handled := true
	switch event.Key {
	case termbox.KeyF1: //sort

		widgets.Stacks.Sort()
	case termbox.KeyF5: // refresh
		h.dry.appmessage("Refreshing stack list")
		h.widget.Unmount()

	case termbox.KeyEnter: //inspect
		showTasks := func(stack string) error {
			widgets.StackTasks.ForStack(stack)
			h.dry.SetViewMode(StackTasks)
			return refreshScreen()
		}
		h.widget.OnEvent(showTasks)
	case termbox.KeyCtrlR: //remove stack
		rw := appui.NewPrompt("The selected stack will be removed. Do you want to proceed? y/N")
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
			removeStack := func(stack string) error {
				err := h.dry.dockerDaemon.StackRemove(stack)
				if err == nil {
					h.dry.appmessage(fmt.Sprintf("Stack %s removed", stack))
				}
				refreshScreen()
				return err
			}
			if err := h.widget.OnEvent(removeStack); err != nil {
				h.dry.appmessage("There was an error removing the stack: " + err.Error())
			}
		}()
	default:
		handled = false
	}
	if !handled {
		switch event.Ch {
		case '6':
			//already in stack screen
			handled = true
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
	}
	if handled {
		refreshScreen()
	} else {
		h.baseEventHandler.handle(event, f)
	}
}
