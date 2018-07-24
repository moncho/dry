package app

import (
	"github.com/moncho/dry/appui"
	termbox "github.com/nsf/termbox-go"
)

type monitorScreenEventHandler struct {
	baseEventHandler
	widget *appui.Monitor
}

func (h *monitorScreenEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	handled := false
	cursor := h.screen.Cursor
	switch event.Key {
	case termbox.KeyArrowUp: //cursor up
		handled = true
		cursor.ScrollCursorUp()
		h.widget.OnEvent(nil)
	case termbox.KeyArrowDown: // cursor down
		handled = true
		cursor.ScrollCursorDown()
		h.widget.OnEvent(nil)
	case termbox.KeyEnter: //Container menu
		showMenu := func(id string) error {
			h.widget.Unmount()
			h.screen.Cursor.Reset()
			widgets.ContainerMenu.ForContainer(id)
			widgets.ContainerMenu.OnUnmount = func() error {
				h.screen.Cursor.Reset()
				h.dry.SetViewMode(Monitor)
				f(h)
				return refreshScreen()
			}
			h.dry.SetViewMode(ContainerMenu)
			f(viewsToHandlers[ContainerMenu])
			return refreshScreen()
		}
		if err := h.widget.OnEvent(showMenu); err != nil {
			h.dry.appmessage(err.Error())
		}
	}
	if !handled {
		switch event.Ch {
		case 'g': //Cursor to the top
			handled = true
			cursor.Reset()
			h.widget.OnEvent(nil)

		case 'G': //Cursor to the bottom
			handled = true
			cursor.Bottom()
			h.widget.OnEvent(nil)
		default:
			handled = false
		}
	}
	if !handled {
		nh := func(eh eventHandler) {
			if cancelMonitorWidget != nil {
				cancelMonitorWidget()
				cancelMonitorWidget = nil
			}
			f(eh)
		}
		h.baseEventHandler.handle(event, nh)
	}
}
