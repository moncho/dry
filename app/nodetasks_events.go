package app

import (
	"github.com/moncho/dry/appui"
	termbox "github.com/nsf/termbox-go"
)

type taskScreenEventHandler struct {
	baseEventHandler
}

func (h *taskScreenEventHandler) widget() appui.AppWidget {
	return h.dry.widgetRegistry.NodeTasks
}

func (h *taskScreenEventHandler) handle(event termbox.Event) {
	if h.passingEvents {
		h.eventChan <- event
		return
	}
	handled := false
	switch event.Key {
	case termbox.KeyEsc:
		handled = true
		h.dry.ShowNodes()
	case termbox.KeyF1: //sort
		handled = true
		h.dry.widgetRegistry.NodeTasks.Sort()
	case termbox.KeyF5: // refresh
		h.widget().Unmount()
		handled = true
	}
	if !handled {
		switch event.Ch {
		case '%':
			handled = true
			showFilterInput(h)
		}
	}
	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(true)
		refreshScreen()
	}

}
