package app

import (
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//KeyPressEvent maps a key to an app action
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
	if event.Key == termbox.KeyArrowUp { //cursor up
		screen.ScrollCursorUp()
		refresh = true
	} else if event.Key == termbox.KeyArrowDown { // cursor down
		screen.ScrollCursorDown()
		refresh = true
	} else if event.Key == termbox.KeyF1 { //sort
		dry.Sort()
	} else if event.Key == termbox.KeyF2 { //show all containers
		dry.ToggleShowAllContainers()
	} else if event.Key == termbox.KeyF5 { // refresh
		dry.Refresh()
	} else if event.Key == termbox.KeyF10 { // docker info
		dry.ShowInfo()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	} else if event.Ch == '?' || event.Ch == 'h' || event.Ch == 'H' { //help
		focus = false
		dry.ShowHelp()
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	} else if event.Ch == '1' {
		screen.Cursor.Line = 0
		dry.ShowImages()
	} else if event.Ch == '2' {
		screen.Cursor.Line = 0
		dry.ShowNetworks()
	} else if event.Ch == 'e' || event.Ch == 'E' { //remove
		dry.Rm(screen.CursorPosition())
	} else if event.Key == termbox.KeyCtrlE { //remove all stopped
		dry.RemoveAllStoppedContainers()
	} else if event.Key == termbox.KeyCtrlK { //kill
		dry.Kill(screen.CursorPosition())
	} else if event.Ch == 'l' || event.Ch == 'L' { //logs
		if logs, err := dry.Logs(screen.CursorPosition()); err == nil {
			focus = false
			dry.ShowContainers()
			go stream(screen, logs, h.keyboardQueueForView, h.viewClosed)
		}
	} else if event.Ch == 'r' || event.Ch == 'R' { //start
		dry.StartContainer(screen.CursorPosition())
	} else if event.Ch == 's' || event.Ch == 'S' { //stats
		done, errC, err := dry.Stats(screen.CursorPosition())
		if err == nil {
			focus = false
			go autorefresh(dry, screen, h.keyboardQueueForView, h.viewClosed, done, errC)
		}
	} else if event.Key == termbox.KeyCtrlT { //stop
		dry.StopContainer(screen.CursorPosition())
	} else if event.Key == termbox.KeyEnter { //inspect
		dry.Inspect(screen.CursorPosition())
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
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
	if event.Key == termbox.KeyArrowUp { //cursor up
		screen.ScrollCursorUp()
		refresh = true
	} else if event.Key == termbox.KeyArrowDown { // cursor down
		screen.ScrollCursorDown()
		refresh = true
	} else if event.Key == termbox.KeyF1 { //sort
		dry.SortImages()
	} else if event.Key == termbox.KeyF5 { // refresh
		dry.Refresh()
	} else if event.Key == termbox.KeyF10 { // docker info
		dry.ShowInfo()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	} else if event.Ch == '?' || event.Ch == 'h' || event.Ch == 'H' { //help
		focus = false
		dry.ShowHelp()
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	} else if event.Ch == '1' {
		screen.Cursor.Line = 0
		dry.ShowContainers()
	} else if event.Ch == '2' {
		screen.Cursor.Line = 0
		dry.ShowNetworks()
	} else if event.Key == termbox.KeyCtrlE { //remove image
		dry.RemoveImage(screen.CursorPosition())
	} else if event.Ch == 'i' || event.Ch == 'I' { //image history
		dry.History(screen.CursorPosition())
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	} else if event.Key == termbox.KeyEnter { //inspect image
		dry.InspectImage(screen.CursorPosition())
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
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
	if event.Key == termbox.KeyArrowUp { //cursor up
		screen.ScrollCursorUp()
		refresh = true
	} else if event.Key == termbox.KeyArrowDown { // cursor down
		screen.ScrollCursorDown()
		refresh = true
	} else if event.Key == termbox.KeyF1 { //sort
		dry.SortNetworks()
	} else if event.Key == termbox.KeyF5 { // refresh
		dry.Refresh()
	} else if event.Key == termbox.KeyF10 { // docker info
		dry.ShowInfo()
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	} else if event.Ch == '?' || event.Ch == 'h' || event.Ch == 'H' { //help
		focus = false
		dry.ShowHelp()
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
	} else if event.Ch == '1' {
		screen.Cursor.Line = 0
		dry.ShowContainers()
	} else if event.Ch == '2' {
		screen.Cursor.Line = 0
		dry.ShowImages()
	} else if event.Key == termbox.KeyEnter { //inspect
		dry.InspectNetwork(screen.CursorPosition())
		focus = false
		go less(dry, screen, h.keyboardQueueForView, h.viewClosed)
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
