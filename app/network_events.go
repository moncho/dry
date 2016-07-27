package app

import (
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type networksScreenEventHandler struct {
	dry                  *Dry
	screen               *ui.Screen
	keyboardQueueForView chan termbox.Event
	viewClosed           chan struct{}
}

func (h networksScreenEventHandler) handle(event termbox.Event) (refresh bool, focus bool) {
	focus = true
	dry := h.dry
	screen := h.screen
	cursor := screen.Cursor
	cursorPos := cursor.Position()
	switch event.Key {
	case termbox.KeyArrowUp: //cursor up
		cursor.ScrollCursorUp()
		refresh = true
	case termbox.KeyArrowDown: // cursor down
		cursor.ScrollCursorDown()
		refresh = true
	case termbox.KeyF1: //sort
		dry.SortNetworks()
	case termbox.KeyF5: // refresh
		cursor.Reset()
		dry.Refresh()
	case termbox.KeyF9: // docker events
		dry.ShowDockerEvents()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	case termbox.KeyF10: // docker info
		dry.ShowInfo()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	case termbox.KeyEnter: //inspect
		dry.InspectNetworkAt(cursorPos)
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	}

	switch event.Ch {
	case '?', 'h', 'H': //help
		focus = false
		dry.ShowHelp()
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	case '1':
		cursor.Reset()
		dry.ShowContainers()
	case '2':
		cursor.Reset()
		dry.ShowImages()
	}
	return (refresh || dry.Changed()), focus
}
