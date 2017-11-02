package app

import (
	"github.com/moncho/dry/appui"
	"github.com/nsf/termbox-go"
)

type stackTasksScreenEventHandler struct {
	baseEventHandler
}

func (h *stackTasksScreenEventHandler) widget() appui.EventableWidget {
	return h.dry.widgetRegistry.StackTasks
}

func (h *stackTasksScreenEventHandler) handle(event termbox.Event) {

	handled := false

	switch event.Key {
	case termbox.KeyEsc:
		handled = true
		h.dry.ShowStacks()
	case termbox.KeyF1: //sort
		handled = true
		h.dry.widgetRegistry.StackTasks.Sort()
	case termbox.KeyF5: // refresh
		h.widget().Unmount()
		handled = true
	}
	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(true)
		refreshScreen()
	}

}
