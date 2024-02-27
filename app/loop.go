package app

import (
	"sync"
	"time"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/ui"
	log "github.com/sirupsen/logrus"
)

var refreshScreen func() error
var refreshIfView func(v viewMode) error
var widgets *widgetRegistry

type nextHandler func(eh eventHandler)

// RenderLoop runs dry
// nolint: gocyclo
func RenderLoop(dry *Dry) {
	if ok, err := dry.Ok(); !ok {
		log.Error(err.Error())
		return
	}
	screen := dry.screen
	termuiEvents, done := ui.EventChannel()

	//use to signal rendering
	renderChan := make(chan struct{})

	var closingLock sync.RWMutex
	refreshScreen = func() error {
		closingLock.RLock()
		defer closingLock.RUnlock()

		renderChan <- struct{}{}
		return nil
	}

	refreshIfView = func(v viewMode) error {
		if v == dry.viewMode() {
			return refreshScreen()
		}
		return nil
	}

	dryOutputChan := dry.OuputChannel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

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

	handler := viewsToHandlers[dry.viewMode()]
	//main loop that handles termui events
loop:
	for event := range termuiEvents {
		switch ev := event.(type) {
		case *tcell.EventInterrupt:
			break loop
		case *tcell.EventKey:
			//Ctrl+C breaks the loop (and exits dry) no matter what
			if ev.Key() == tcell.KeyCtrlC || ev.Rune() == 'Q' {
				break loop
			}
			handler.handle(ev, func(eh eventHandler) {
				handler = eh
			})

		case *tcell.EventResize:
			screen.Resize()
			//Reload dry ui elements
			//TODO widgets.reload()
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
