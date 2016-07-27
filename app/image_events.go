package app

import (
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type imagesScreenEventHandler struct {
	dry                  *Dry
	screen               *ui.Screen
	keyboardQueueForView chan termbox.Event
	viewClosed           chan struct{}
}

func (h imagesScreenEventHandler) handle(event termbox.Event) (refresh bool, focus bool) {
	focus = true
	dry := h.dry
	screen := h.screen
	cursor := screen.Cursor
	cursorPos := cursor.Position()
	//Controls if the event has been handled by the first switch statement
	handled := true

	switch event.Key {
	case termbox.KeyArrowUp: //cursor up
		cursor.ScrollCursorUp()
		refresh = true
	case termbox.KeyArrowDown: // cursor down
		cursor.ScrollCursorDown()
		refresh = true
	case termbox.KeyF1: //sort
		dry.SortImages()
	case termbox.KeyF5: // refresh
		dry.Refresh()
	case termbox.KeyF9: // docker events
		dry.ShowDockerEvents()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	case termbox.KeyF10: // docker info
		dry.ShowInfo()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	case termbox.KeyCtrlD: //remove dangling images
		dry.RemoveDanglingImages()
		cursor.ScrollCursorDown()
	case termbox.KeyCtrlE: //remove image
		dry.RemoveImageAt(cursorPos, false)
		cursor.ScrollCursorDown()
	case termbox.KeyCtrlF: //force remove image
		dry.RemoveImageAt(cursorPos, true)
		cursor.ScrollCursorDown()
	case termbox.KeyEnter: //inspect image
		dry.InspectImageAt(cursorPos)
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	default:
		handled = false
	}

	if !handled {
		switch event.Ch {
		case '?', 'h', 'H': //help
			focus = false
			dry.ShowHelp()
			go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
		case '1':
			cursor.Reset()
			dry.ShowContainers()
		case '3':
			cursor.Reset()
			dry.ShowNetworks()
		case 'i', 'I': //image history
			dry.HistoryAt(cursorPos)
			focus = false
			go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
		}

	}

	return (refresh || dry.Changed()), focus
}
