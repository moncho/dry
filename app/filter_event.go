package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

func showFilterInput(eh eventHandler) {
	rw := appui.NewAskForConfirmation("Filter? (blank to remove current filter)")
	eh.setForwardEvents(true)
	eh.widgetRegistry().add(rw)
	go func() {
		events := ui.EventSource{
			Events: eh.getEventChan(),
			EventHandledCallback: func(e termbox.Event) error {
				return refreshScreen()
			},
		}
		rw.OnFocus(events)
		eh.widgetRegistry().remove(rw)
		filter, canceled := rw.Text()
		eh.setForwardEvents(false)

		if canceled {
			return
		}
		eh.widget().Filter(filter)
	}()
}
