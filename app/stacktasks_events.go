package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

type stackTasksScreenEventHandler struct {
	baseEventHandler
}

func (h *stackTasksScreenEventHandler) widget() appui.AppWidget {
	return h.dry.widgetRegistry.StackTasks
}

func (h *stackTasksScreenEventHandler) handle(event termbox.Event) {
	if h.passingEvents {
		h.eventChan <- event
		return
	}
	handled := false

	switch event.Key {
	case termbox.KeyEsc:
		handled = true
		h.dry.ShowStacks()
	case termbox.KeyF1: //sort
		handled = true
		h.widget().Sort()
	case termbox.KeyF5: // refresh
		handled = true
		h.dry.appmessage("Refreshing stack tasks list")
		h.widget().Unmount()
	}
	switch event.Ch {
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
			h.widget().Filter(filter)
		}()
	}

	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(true)
		refreshScreen()
	}

}
