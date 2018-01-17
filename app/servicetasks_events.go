package app

import (
	"fmt"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

type serviceTasksScreenEventHandler struct {
	baseEventHandler
}

func (h *serviceTasksScreenEventHandler) widget() appui.AppWidget {
	return h.dry.widgetRegistry.ServiceTasks
}

func (h *serviceTasksScreenEventHandler) handle(event termbox.Event) {
	if h.forwardingEvents {
		h.eventChan <- event
		return
	}
	handled := false
	focus := true

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
	case termbox.KeyEnter:
		handled = true
		focus = false
		if err := h.widget().OnEvent(inspectTask(h.dry, h.screen, h.eventChan, h.closeViewChan)); err != nil {
			h.dry.appmessage(
				fmt.Sprintf("Error inspecting stack: %s", err.Error()))
		}

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
		h.setFocus(focus)
		refreshScreen()
	}

}

func inspectTask(
	dry *Dry,
	screen *ui.Screen,
	eventChan chan termbox.Event,
	closeViewChan chan struct{}) func(id string) error {
	return func(id string) error {
		stack, err := dry.dockerDaemon.Task(id)
		if err != nil {
			return err
		}
		renderer := appui.NewJSONRenderer(stack)
		go appui.Less(renderer, screen, eventChan, closeViewChan)
		return nil
	}
}
