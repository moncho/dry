package app

import (
	"fmt"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

type stacksScreenEventHandler struct {
	baseEventHandler
}

func (h *stacksScreenEventHandler) widget() appui.AppWidget {
	return h.dry.widgetRegistry.Stacks
}

func (h *stacksScreenEventHandler) handle(event termbox.Event) {
	if h.forwardingEvents {
		h.eventChan <- event
		return
	}
	focus := true
	handled := false
	switch event.Key {
	case termbox.KeyF1: //sort
		handled = true
		h.dry.widgetRegistry.Stacks.Sort()
	case termbox.KeyF5: // refresh
		handled = true
		h.dry.appmessage("Refreshing stack list")
		h.widget().Unmount()

	case termbox.KeyEnter: //inspect
		handled = true
		showTasks := func(stack string) error {
			h.dry.ShowStackTasks(stack)
			return refreshScreen()
		}
		h.widget().OnEvent(showTasks)
	case termbox.KeyCtrlR: //remove stack
		rw := appui.NewPrompt("The selected stack will be removed. Do you want to proceed? y/N")
		h.setForwardEvents(true)
		handled = true
		h.dry.widgetRegistry.add(rw)
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			rw.OnFocus(events)
			h.dry.widgetRegistry.remove(rw)
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
			if err := h.widget().OnEvent(removeStack); err != nil {
				h.dry.appmessage("There was an error removing the stack: " + err.Error())
			}
		}()
	}
	if !handled {
		switch event.Ch {
		case '6':
			//already in stack screen
			handled = true
		case '%':
			handled = true
			showFilterInput(h)
		}
	}
	if handled {
		h.setFocus(focus)
		if h.hasFocus() {
			refreshScreen()
		}
	} else {
		h.baseEventHandler.handle(event)
	}
}
