package app

import (
	"github.com/moncho/dry/appui"
	"github.com/nsf/termbox-go"
)

type imagesScreenEventHandler struct {
	baseEventHandler
}

func (h *imagesScreenEventHandler) handle(event termbox.Event) {
	focus := true
	dry := h.dry
	screen := h.screen
	cursor := screen.Cursor
	cursorPos := cursor.Position()
	//Controls if the event has been handled by the first switch statement
	handled := true

	switch event.Key {
	case termbox.KeyF1: //sort
		dry.SortImages()
	case termbox.KeyCtrlD: //remove dangling images
		dry.RemoveDanglingImages()
	case termbox.KeyCtrlE: //remove image
		dry.RemoveImageAt(cursorPos, false)
		cursor.ScrollCursorDown()
	case termbox.KeyCtrlF: //force remove image
		dry.RemoveImageAt(cursorPos, true)
		cursor.ScrollCursorDown()
	case termbox.KeyEnter: //inspect image
		dry.InspectImageAt(cursorPos)
		focus = false
		go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.closeViewChan)
	default:
		handled = false
	}

	if !handled {
		switch event.Ch {
		case '2':
			handled = true

		case 'i', 'I': //image history
			handled = true

			dry.HistoryAt(cursorPos)
			focus = false
			go appui.Less(renderDry(dry), screen, h.keyboardQueueForView, h.closeViewChan)
		}

	}
	if handled {
		h.setFocus(focus)
		if h.hasFocus() {
			h.renderChan <- struct{}{}
		}
	} else {
		h.baseEventHandler.handle(event)
	}
}
