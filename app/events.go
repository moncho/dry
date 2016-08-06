package app

import (
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//eventHandler maps a key to an app action, returned value
//tells if the main screen should forward keypresses to
//view channel or not
type eventHandler interface {
	handle(renderChan chan<- struct{}, event termbox.Event) (dontForwardEvents bool)
}

//eventHandlerFactory creates eventHandlers
func eventHandlerFactory(
	dry *Dry,
	screen *ui.Screen,
	keyboardQueueForView chan termbox.Event,
	viewClosed chan struct{}) eventHandler {
	switch dry.viewMode() {
	case Images:
		return imagesScreenEventHandler{dry, screen, keyboardQueueForView, viewClosed}
	case Networks:
		return networksScreenEventHandler{dry, screen, keyboardQueueForView, viewClosed}
		/*	case ContainerCommandsMode:
			return containersCommandsEventHandler{dry, screen, keyboardQueueForView, viewClosed}*/
	default:
		return containersScreenEventHandler{dry, screen, keyboardQueueForView, viewClosed}
	}
}
