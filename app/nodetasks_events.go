package app

import (
	"github.com/moncho/dry/appui"
	"github.com/nsf/termbox-go"
)

type taskScreenEventHandler struct {
	baseEventHandler
}

func (h *taskScreenEventHandler) widget() appui.EventableWidget {
	return h.dry.widgetRegistry.NodeTasks
}

func (h *taskScreenEventHandler) handle(event termbox.Event) {

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
		h.baseEventHandler.handle(event)
	} else {
		refreshScreen()
	}

}
