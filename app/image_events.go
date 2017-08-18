package app

import (
	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type imagesScreenEventHandler struct {
	baseEventHandler
	passingEvents bool
}

func (h *imagesScreenEventHandler) handle(event termbox.Event) {
	if h.passingEvents {
		h.eventChan <- event
		return
	}
	//Controls if the event has been handled by the first switch statement
	var handled bool
	var keepFocus bool

	handled, keepFocus = h.handleKeyEvent(event.Key)

	if !handled {
		handled, keepFocus = h.handleChEvent(event.Ch)
	}
	if handled {
		h.setFocus(keepFocus)
		if h.hasFocus() {
			h.renderChan <- struct{}{}
		}
	} else {
		h.baseEventHandler.handle(event)
	}
}

func (h *imagesScreenEventHandler) handleKeyEvent(key termbox.Key) (bool, bool) {
	dry := h.dry
	screen := h.screen
	cursor := screen.Cursor
	cursorPos := cursor.Position()
	keepFocus := true
	handled := true
	switch key {
	case termbox.KeyF1: //sort
		dry.SortImages()
	case termbox.KeyCtrlD: //remove dangling images
		dry.RemoveDanglingImages()
	case termbox.KeyCtrlE: //remove image
		dry.RemoveImageAt(cursorPos, false)
	case termbox.KeyCtrlF: //force remove image
		dry.RemoveImageAt(cursorPos, true)
	case termbox.KeyEnter: //inspect image
		dry.InspectImageAt(cursorPos)
		keepFocus = false
		go appui.Less(renderDry(dry), screen, h.eventChan, h.closeViewChan)
	default:
		handled = false
	}
	return handled, keepFocus
}

func (h *imagesScreenEventHandler) handleChEvent(ch rune) (bool, bool) {
	dry := h.dry
	screen := h.screen
	cursor := screen.Cursor
	cursorPos := cursor.Position()
	keepFocus := true
	handled := true
	switch ch {
	case '2': //Ignore since dry is already on the images screen

	case 'i', 'I': //image history
		dry.HistoryAt(cursorPos)
		keepFocus = false
		go appui.Less(renderDry(dry), screen, h.eventChan, h.closeViewChan)

	case 'r', 'R': //Run container
		if image, err := dry.dockerDaemon.ImageAt(cursorPos); err == nil {
			rw := appui.NewImageRunWidget(image)
			h.passingEvents = true
			dry.widgetRegistry.add(rw)
			go func(image *types.ImageSummary) {
				events := ui.EventSource{
					Events: h.eventChan,
					EventHandledCallback: func(e termbox.Event) error {
						h.renderChan <- struct{}{}
						return nil
					},
				}
				rw.OnFocus(events)
				dry.widgetRegistry.remove(rw)
				runCommand := rw.Text()
				h.passingEvents = false
				if err := dry.dockerDaemon.RunImage(image, runCommand); err != nil {
					dry.appmessage(err.Error())
				}

			}(image)
		}
	default:
		handled = false

	}
	return handled, keepFocus
}
