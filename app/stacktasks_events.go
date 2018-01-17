package app

import (
	"fmt"

	"github.com/moncho/dry/appui"
	termbox "github.com/nsf/termbox-go"
)

type stackTasksScreenEventHandler struct {
	baseEventHandler
}

func (h *stackTasksScreenEventHandler) widget() appui.AppWidget {
	return h.dry.widgetRegistry.StackTasks
}

func (h *stackTasksScreenEventHandler) handle(event termbox.Event) {
	if h.forwardingEvents {
		h.eventChan <- event
		return
	}
	handled := false
	focus := true

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
	case termbox.KeyEnter:
		handled = true
		focus = false
		if err := h.widget().OnEvent(inspectTask(h.dry, h.screen, h.eventChan, h.closeViewChan)); err != nil {
			h.dry.appmessage(
				fmt.Sprintf("Error inspecting stack: %s", err.Error()))
		}

	}
	switch event.Ch {
	case '%':
		handled = true
		showFilterInput(h)
	}

	if !handled {
		h.baseEventHandler.handle(event)
	} else {
		h.setFocus(focus)
		refreshScreen()
	}

}
