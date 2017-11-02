package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type stacksScreenEventHandler struct {
	baseEventHandler
	passingEvents bool
}

func (h *stacksScreenEventHandler) widget() appui.EventableWidget {
	return h.dry.widgetRegistry.Stacks
}

func (h *stacksScreenEventHandler) handle(event termbox.Event) {
	if h.passingEvents {
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

	}
	if !handled {
		switch event.Ch {
		case '6':
			//already in stack screen
			handled = true
		case '%':
			rw := appui.NewAskForConfirmation("Filter? (blank to remove current filter)")
			h.passingEvents = true
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
				filter, canceled := rw.Text()
				h.passingEvents = false
				if canceled {
					return
				}
				h.dry.widgetRegistry.Stacks.Filter(filter)
			}()
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
