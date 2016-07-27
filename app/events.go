package app

import (
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//eventHandler maps a key to an app action
type eventHandler interface {
	handle(event termbox.Event) (refresh bool, focus bool)
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
