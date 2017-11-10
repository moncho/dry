package app

import (
	"fmt"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

type nodesScreenEventHandler struct {
	baseEventHandler
}

func (h *nodesScreenEventHandler) widget() appui.AppWidget {
	return h.dry.widgetRegistry.Nodes
}

func (h *nodesScreenEventHandler) handle(event termbox.Event) {
	if h.passingEvents {
		h.eventChan <- event
		return
	}
	handled := false
	focus := true

	switch event.Key {
	case termbox.KeyF1: //sort
		handled = true
		h.dry.widgetRegistry.Nodes.Sort()
	case termbox.KeyF5: // refresh
		h.widget().Unmount()
		handled = true
	case termbox.KeyCtrlA:
		dry := h.dry
		rw := appui.NewAskForConfirmation("Changing node availability, please type one of ('active'|'pause'|'drain')")
		h.passingEvents = true
		handled = true
		dry.widgetRegistry.add(rw)
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			rw.OnFocus(events)
			dry.widgetRegistry.remove(rw)
			availability, canceled := rw.Text()
			h.passingEvents = false
			if canceled {
				return
			}
			if availability != "active" && availability != "pause" && availability != "drain" {
				dry.appmessage(fmt.Sprintf("Invalid availability: %s", availability))
				return
			}

			changeNode := func(nodeID string) error {
				err := dry.dockerDaemon.NodeChangeAvailability(
					nodeID,
					docker.NewNodeAvailability(availability))

				if err == nil {
					dry.appmessage(fmt.Sprintf("Node %s availability is now %s", nodeID, availability))
				} else {
					dry.appmessage(fmt.Sprintf("Could not change node availability, error %s", err.Error()))
					return err
				}
				return refreshScreen()
			}
			h.widget().OnEvent(changeNode)
		}()

	case termbox.KeyEnter:
		showServices := func(nodeID string) error {
			h.screen.Cursor.Reset()
			h.dry.ShowTasks(nodeID)
			return refreshScreen()
		}
		h.widget().OnEvent(showServices)
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
		h.setFocus(focus)
		if h.hasFocus() {
			refreshScreen()
		}
	}
}
