package app

import (
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

type nodesScreenEventHandler struct {
	baseEventHandler
	widget *swarm.NodesWidget
}

func (h *nodesScreenEventHandler) handle(event *tcell.EventKey, f func(eventHandler)) {

	handled := false

	switch event.Key() {
	case tcell.KeyF1: //sort
		handled = true
		widgets.Nodes.Sort()
	case tcell.KeyF5: // refresh
		h.widget.Unmount()
		handled = true
	case tcell.KeyCtrlA:
		dry := h.dry
		rw := appui.NewPrompt("Changing node availability, please type one of ('active'|'pause'|'drain')")
		forwarder := newRegisteredEventForwarder(f)
		refreshScreen()
		handled = true
		widgets.add(rw)
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
					return refreshScreen()
				},
			}
			rw.OnFocus(events)
			widgets.remove(rw)
			availability, canceled := rw.Text()
			f(h)
			if canceled {
				return
			}
			if availability != "active" && availability != "pause" && availability != "drain" {
				dry.message(fmt.Sprintf("Invalid availability: %s", availability))
				return
			}

			changeNode := func(nodeID string) error {
				err := dry.dockerDaemon.NodeChangeAvailability(
					nodeID,
					docker.NewNodeAvailability(availability))

				if err == nil {
					dry.message(fmt.Sprintf("Node %s availability is now %s", nodeID, availability))
				} else {
					dry.message(fmt.Sprintf("Could not change node availability, error %s", err.Error()))
					return err
				}
				return refreshScreen()
			}
			h.widget.OnEvent(changeNode)
		}()

	case tcell.KeyEnter:
		showServices := func(nodeID string) error {
			h.screen.Cursor().Reset()
			widgets.NodeTasks.ForNode(nodeID)
			h.dry.changeView(Tasks)
			f(viewsToHandlers[Tasks])
			return refreshScreen()
		}
		h.widget.OnEvent(showServices)
		handled = true
	}
	if !handled {
		switch event.Rune() {
		case '%':
			handled = true
			forwarder := newRegisteredEventForwarder(f)
			applyFilter := func(filter string, canceled bool) {
				if !canceled {
					h.widget.Filter(filter)
				}
				f(h)
			}
			showFilterInput(newEventSource(forwarder.events()), applyFilter)
		}
	}
	if !handled {
		h.baseEventHandler.handle(event, f)
	} else {
		refreshScreen()
	}
}
