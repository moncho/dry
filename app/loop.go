package app

import (
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type focusTracker struct {
	mutex sync.Locker
	focus bool
}

func (f *focusTracker) set(b bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.focus = b
}

func (f *focusTracker) hasFocus() bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.focus
}

func (f *focusTracker) flip() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.focus = !f.focus
}

//RenderLoop renders dry until it quits
func RenderLoop(dry *Dry, screen *ui.Screen) {
	if ok, _ := dry.Ok(); !ok {
		return
	}

	termuiEvents, done := ui.EventChannel()
	keyboardQueue := make(chan termbox.Event)

	timer := time.NewTicker(1 * time.Second)

	viewClosed := make(chan struct{})
	//On receive dry is rendered
	renderChan := make(chan struct{}, 1)

	keyboardQueueForView := make(chan termbox.Event)
	dryOutputChan := dry.OuputChannel()
	statusBar := ui.NewStatusBar(0, ui.ActiveScreen.Dimensions.Width, appui.DryTheme)
	eventHandlerFactory := &eventHandlerFactory{
		dry:                  dry,
		screen:               screen,
		keyboardQueueForView: keyboardQueueForView,
		viewClosed:           viewClosed,
		renderChan:           renderChan}

	defer timer.Stop()
	defer close(done)
	defer close(keyboardQueue)
	defer close(keyboardQueueForView)
	defer close(viewClosed)
	defer close(renderChan)

	//tracks if the main loop has the focus (and responds to events),
	//or if events have to be delegated.
	//creation belongs outside the loop
	focus := &focusTracker{&sync.Mutex{}, true}

	//renders dry on message until renderChan is closed
	go func() {
		for {
			_, ok := <-renderChan
			if ok {
				screen.Clear()
				Render(dry, screen, statusBar)
			} else {
				return
			}
		}
	}()

	renderChan <- struct{}{}

	//timer and status bar are shown if the main loop has the focus
	go func(focus *focusTracker) {
		for {
			select {
			case <-timer.C:
				if focus.hasFocus() {
					timestamp := time.Now().Format(`15:04:05`)
					screen.RenderLine(0, 0, `<right><white>`+timestamp+`</></right>`)
					screen.Flush()
				}
			case dryMessage, ok := <-dryOutputChan:
				if ok {
					if focus.hasFocus() {
						statusBar.StatusMessage(dryMessage, 10*time.Second)
						if dry.Changed() {
							renderChan <- struct{}{}
						} else {
							statusBar.Render()
						}
					} else {
						//stop the status bar until the focus is retrieved
						statusBar.Stop()
					}
				} else {
					return
				}
			}
		}
	}(focus)

	go func() {
		for range viewClosed {
			focus.flip()
			dry.ShowMainView()
			renderChan <- struct{}{}
		}
	}()

	go func() {
		for event := range keyboardQueue {
			if focus.hasFocus() {
				handler := eventHandlerFactory.handlerFor(dry.viewMode())
				if handler != nil {
					handler.handle(event)
					focus.set(handler.hasFocus())
				}
			} else {
				//Whoever has the focus, handles the event
				select {
				case keyboardQueueForView <- event:
				default:
				}
			}
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
			if event.Key == termbox.KeyCtrlC || (focus.hasFocus() && (event.Ch == 'q' || event.Ch == 'Q')) {
				break loop
			} else {
				select {
				case keyboardQueue <- event:
				default:
				}
			}
		case termbox.EventResize:
			ui.Resize()
			//Reload dry ui elements
			dry.ui = appui.NewAppUI(dry.dockerDaemon)
		}
	}

	log.Debug("something broke the loop. Time to die")
}
