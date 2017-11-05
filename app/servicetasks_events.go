package app

import (
	"github.com/moncho/dry/appui"
	termbox "github.com/nsf/termbox-go"
)

type serviceTasksScreenEventHandler struct {
	baseEventHandler
}

func (h *serviceTasksScreenEventHandler) widget() appui.AppWidget {
	return h.dry.widgetRegistry.ServiceTasks
}

func (h *serviceTasksScreenEventHandler) handle(event termbox.Event) {

	handled := false

	switch event.Key {
	case termbox.KeyEsc:
		handled = true
		h.dry.ShowServices()
	case termbox.KeyF1: //sort
		handled = true
		h.dry.widgetRegistry.ServiceTasks.Sort()
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
