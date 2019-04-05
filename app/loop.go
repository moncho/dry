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

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()

		for _ = range renderChan {
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

	handler := viewsToHandlers[dry.viewMode()]
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
				handler.handle(event, func(eh eventHandler) {
					handler = eh
				})
			}
		case termbox.EventResize:
			ui.Resize()
			//Reload dry ui elements
			widgets = newWidgetRegistry(dry.dockerDaemon)
		}
	}

	log.Debug("something broke the loop. Time to die")

	//Close terminal event channel
	close(done)
	//make the global refreshScreen func a noop before closing
	closingLock.Lock()
	refreshScreen = func() error {
		return nil
	}
	closingLock.Unlock()

	//Close the channel used to notify the rendering goroutine
	close(renderChan)
	//Wait for the rendering goroutine to exit
	wg.Wait()
}
