package app

import (
	"fmt"

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
		widgets.add(rw)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			refreshScreen()

			rw.OnFocus(events)
			widgets.remove(rw)
			confirmation, canceled := rw.Text()
			f(h)
			if canceled || (confirmation != "y" && confirmation != "Y") {
				return
			}

			pr, err := h.dry.dockerDaemon.Prune()
			if err == nil {
				if du, err := h.dry.dockerDaemon.DiskUsage(); err == nil {
					widgets.DiskUsage.PrepareToRender(&du, pr)
				}
			} else {
				h.dry.message(
					fmt.Sprintf(
						"<red>Error running prune. %s</>", err))
			}
			refreshScreen()
		}()
	}
	if !handled {
		h.baseEventHandler.handle(event, f)
	}

}
