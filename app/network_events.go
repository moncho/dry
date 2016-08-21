package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type networksScreenEventHandler struct {
	dry                  *Dry
	screen               *ui.Screen
	keyboardQueueForView chan termbox.Event
	viewClosed           chan struct{}
}

func (h networksScreenEventHandler) handle(renderChan chan<- struct{}, event termbox.Event) (focus bool) {
	focus = true
	dry := h.dry
	screen := h.screen
	cursor := screen.Cursor
	cursorPos := cursor.Position()
	switch event.Key {
	case termbox.KeyArrowUp: //cursor up
		cursor.ScrollCursorUp()
	case termbox.KeyArrowDown: // cursor down
		cursor.ScrollCursorDown()
	case termbox.KeyF1: //sort
		dry.SortNetworks()
	case termbox.KeyF5: // refresh
		cursor.Reset()
		dry.Refresh()
	case termbox.KeyF9: // docker events
		dry.ShowDockerEvents()
		focus = false
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.viewClosed)
	case termbox.KeyF10: // docker info
		dry.ShowInfo()
		focus = false
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.viewClosed)
	case termbox.KeyEnter: //inspect
		dry.InspectNetworkAt(cursorPos)
		focus = false
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.viewClosed)
	case termbox.KeyCtrlE: //remove network
		if cursorPos >= 0 {
			network, err := dry.NetworkAt(cursorPos)
			if err == nil {
				dry.RemoveNetwork(network.ID)
				cursor.ScrollCursorDown()
			} else {
				ui.ShowErrorMessage(screen, h.keyboardQueueForView, h.viewClosed, err)
			}
		}
	}

	switch event.Ch {
	case '?', 'h', 'H': //help
		focus = false
		dry.ShowHelp()
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.viewClosed)
	case '1':
		cursor.Reset()
		dry.ShowContainers()
	case '2':
		cursor.Reset()
		dry.ShowImages()
	}
	if focus {
		renderChan <- struct{}{}
	}
	return focus
}
