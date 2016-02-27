package app

import (
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//eventHandler maps a key to an app action
type eventHandler interface {
	handle(event termbox.Event) (refresh bool, focus bool)
}

type containersScreenEventHandler struct {
	dry                  *Dry
	screen               *ui.Screen
	keyboardQueueForView chan termbox.Event
	viewClosed           chan struct{}
}

func (h containersScreenEventHandler) handle(event termbox.Event) (refresh bool, focus bool) {
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
		dry.Sort()
	case termbox.KeyF2: //show all containers
		cursor.Reset()
		dry.ToggleShowAllContainers()
	case termbox.KeyF5: // refresh
		dry.Refresh()
	case termbox.KeyF10: // docker info
		dry.ShowInfo()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	case termbox.KeyCtrlE: //remove all stopped
		dry.RemoveAllStoppedContainers()
	case termbox.KeyCtrlK: //kill
		dry.Kill(cursorPos)
	case termbox.KeyCtrlR: //start
		dry.RestartContainer(cursorPos)
	case termbox.KeyCtrlT: //stop
		dry.StopContainer(cursorPos)
	case termbox.KeyEnter: //inspect
		dry.Inspect(cursorPos)
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	default: //Not handled
		handled = false
	}
	if !handled {
		switch event.Ch {
		case 's', 'S': //stats
			done, errC, err := dry.Stats(cursorPos)
			if err == nil {
				focus = false
				go autorefresh(dry, screen, h.keyboardQueueForView, h.viewClosed, done, errC)
			}
		case 'l', 'L': //logs
			if logs, err := dry.Logs(cursorPos); err == nil {
				focus = false
				dry.ShowContainers()
				go stream(screen, logs, h.keyboardQueueForView, h.viewClosed)
			}
		case '?', 'h', 'H': //help
			focus = false
			dry.ShowHelp()
			go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
		case '1':
			cursor.Reset()
			dry.ShowImages()
		case '2':
			cursor.Reset()
			dry.ShowNetworks()
		case 'e', 'E': //remove
			dry.Rm(cursorPos)
			cursor.ScrollCursorUp()
		}
	}
	return (refresh || dry.Changed()), focus
}

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
	case termbox.KeyF10: // docker info
		dry.ShowInfo()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)

	case termbox.KeyCtrlE: //remove image
		dry.RemoveImage(cursorPos, false)
		cursor.ScrollCursorUp()
	case termbox.KeyCtrlF: //force remove image
		dry.RemoveImage(cursorPos, true)
		cursor.ScrollCursorUp()
	case termbox.KeyEnter: //inspect image
		dry.InspectImage(cursorPos)
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
		case '2':
			cursor.Reset()
			dry.ShowNetworks()
		case 'i', 'I': //image history
			dry.History(cursorPos)
			focus = false
			go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
		}

	}

	return (refresh || dry.Changed()), focus
}

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
	case termbox.KeyF10: // docker info
		dry.ShowInfo()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	case termbox.KeyEnter: //inspect
		dry.InspectNetwork(cursorPos)
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

func eventHandlerFactory(dry *Dry, screen *ui.Screen,
	keyboardQueueForView chan termbox.Event,
	viewClosed chan struct{}) eventHandler {
	switch dry.viewMode() {
	case Images:
		return imagesScreenEventHandler{dry, screen, keyboardQueueForView, viewClosed}
	case Networks:
		return networksScreenEventHandler{dry, screen, keyboardQueueForView, viewClosed}
	default:
		return containersScreenEventHandler{dry, screen, keyboardQueueForView, viewClosed}
	}
}
