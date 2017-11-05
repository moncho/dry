package app

import (
	"github.com/moncho/dry/appui"
	termbox "github.com/nsf/termbox-go"
)

type stacksScreenEventHandler struct {
	baseEventHandler
}

func (h *stacksScreenEventHandler) widget() appui.AppWidget {
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
