package app

import (
	"fmt"
	"time"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

const (
	confirmation = `WARNING! This will remove all unused data. Are you sure you want to continue? [y/N]`
)

type diskUsageScreenEventHandler struct {
	baseEventHandler
}

func (h *diskUsageScreenEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	if h.forwardingEvents() {
		h.eventChan <- event
		return
	}
	handled := false
	switch event.Key {
	case termbox.KeyArrowUp | termbox.KeyArrowDown:
		//To avoid the base handler handling this
		handled = true
	}
	switch event.Ch {
	case 'p', 'P':
		handled = true

		rw := appui.NewPrompt(confirmation)
		h.setForwardEvents(true)
		widgets.add(rw)
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			refreshScreen()

			rw.OnFocus(events)
			widgets.remove(rw)
			confirmation, canceled := rw.Text()
			h.setForwardEvents(false)
			if canceled || (confirmation != "y" && confirmation != "Y") {
				return
			}
			pr, err := h.dry.dockerDaemon.Prune()
			if err == nil {
				h.dry.cache.Add(pruneReport, pr, 30*time.Second)
				refreshScreen()
			} else {
				h.dry.appmessage(
					fmt.Sprintf(
						"<red>Error running prune. %s</>", err))
			}

		}()
	}
	if !handled {
		h.baseEventHandler.handle(event, f)
	}

}
