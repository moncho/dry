package app

import (
	"sync"
	"time"

	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
	log "github.com/sirupsen/logrus"
)

var refreshScreen func() error
var widgets *widgetRegistry

type nextHandler func(eh eventHandler)

//RenderLoop renders dry until it quits
// nolint: gocyclo
func RenderLoop(dry *Dry, screen *ui.Screen) {
	if ok, _ := dry.Ok(); !ok {
		return
	}
	termuiEvents, done := ui.EventChannel()
	eventChan := make(chan termbox.Event)

	//On receive dry is rendered
	renderChan := make(chan struct{})

	var closingLock sync.RWMutex
	refreshScreen = func() error {
		closingLock.RLock()
		defer closingLock.RUnlock()
		renderChan <- struct{}{}
		return nil
	}

	dryOutputChan := dry.OuputChannel()

	defer close(done)
	defer close(eventChan)
	//make the global refreshScreen a noop before closing
	defer func() {
		closingLock.Lock()
		defer closingLock.Unlock()
		refreshScreen = func() error {
			return nil
		}
	}()

	defer close(renderChan)

	//renders dry on message until renderChan is closed
	go func() {
		for range renderChan {
			if !screen.Closing() {
				screen.Clear()
				render(dry, screen)
			}
		}
	}()

	refreshScreen()

	go func() {
		statusBar := widgets.MessageBar
		for {
			select {
			case dryMessage, ok := <-dryOutputChan:
				if ok {
					statusBar.Message(dryMessage, 10*time.Second)
					statusBar.Render()
				} else {
					return
				}
			}
		}
	}()

	go func() {
		//Initial handler
		handler := viewsToHandlers[dry.viewMode()]

		for event := range eventChan {
			handler.handle(event, func(eh eventHandler) {
				handler = eh
			})
		}
	}()

	//main loop that handles termui events
loop:
	for event := range termuiEvents {
		switch event.Type {
		case termbox.EventInterrupt:
			break loop
		case termbox.EventKey:
			//Ctrl+C breaks the loop (and exits dry) no matter what
			if event.Key == termbox.KeyCtrlC || event.Ch == 'Q' {
				break loop
			} else {
				select {
				case eventChan <- event:
				default:
					log.Debug("Skipping termbox key event, channel is busy")
				}

			}
		case termbox.EventResize:
			ui.Resize()
			//Reload dry ui elements
			widgets = newWidgetRegistry(dry.dockerDaemon)
		}
	}

	log.Debug("something broke the loop. Time to die")
}
