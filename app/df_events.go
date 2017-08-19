package app

import (
	"github.com/moncho/dry/appui"
	termbox "github.com/nsf/termbox-go"
)

const (
	confirmation = `WARNING! This will remove all unused data. Are you sure you want to continue? [y/N]`
)

type diskUsageScreenEventHandler struct {
	baseEventHandler
}

func (h *diskUsageScreenEventHandler) handle(event termbox.Event) {
	handled := false
	ignored := false
	switch event.Key {
	case termbox.KeyArrowUp | termbox.KeyArrowDown:
		//To avoid the base handler handling this
		ignored = true
		handled = true
	}
	switch event.Ch {
	case 'p', 'P':
		handled = true
		if confirmation, err := appui.ReadLine(confirmation); err == nil {
			h.screen.ClearAndFlush()
			if confirmation == "Y" || confirmation == "y" {
				h.dry.Prune()
			}
		}
	}
	if handled {
		h.setFocus(true)
		if !ignored {
			refreshScreen()
		}
	} else {
		h.baseEventHandler.handle(event)
	}

}
