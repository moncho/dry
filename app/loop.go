package app

import (
	"io"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/stdcopy"
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

	keyboardQueue, done := ui.EventChannel()
	timestampQueue := time.NewTicker(1 * time.Second)

	viewClosed := make(chan struct{})
	//On receive dry is rendered
	renderChan := make(chan struct{}, 1)

	keyboardQueueForView := make(chan termbox.Event)
	dryOutputChan := dry.OuputChannel()
	statusBar := ui.NewStatusBar(0, screen.Width)

	defer timestampQueue.Stop()
	defer close(done)
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
			select {
			case <-timestampQueue.C:
				timestamp := time.Now().Format(`15:04:05`)
				screen.RenderLine(0, 0, `<right><white>`+timestamp+`</></right>`)
				screen.Flush()
			case _, ok := <-renderChan:
				if ok {
					screen.Clear()
					Render(dry, screen, statusBar)
				} else {
					return
				}
			}
		}
	}()

	renderChan <- struct{}{}

	go func(focus *focusTracker) {
		for {
			dryMessage, ok := <-dryOutputChan
			if ok {
				if focus.hasFocus() {
					statusBar.StatusMessage(dryMessage, 10*time.Second)
					if dry.Changed() {
						renderChan <- struct{}{}
					} else {
						statusBar.Render()
					}
					screen.Flush()
				} else {
					//stop the status bar until the focus is retrieved
					statusBar.Stop()
				}
			} else {
				return
			}
		}
	}(focus)

	go func() {
		for {
			_, ok := <-viewClosed
			if ok {
				focus.flip()
				dry.ShowMainView()
				renderChan <- struct{}{}
			}
		}
	}()

	//loop handles input and timer events until a closing event happens
loop:
	for {
		select {
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventInterrupt:
				break loop
			case termbox.EventKey:
				if event.Key == termbox.KeyCtrlC { //Ctrl+C breaks the loop (and exits dry) no matter what
					break loop
				}
				if focus.hasFocus() {
					if event.Ch == 'q' || event.Ch == 'Q' {
						break loop
					} else {
						handler := eventHandlerFactory(dry, screen, keyboardQueueForView, viewClosed)
						if handler != nil {
							handler.handle(event)
						} else {
							log.Panic("There is no event handler")
						}
					}
				} else {
					//Whoever has the focus, handles the event
					keyboardQueueForView <- event
				}
			case termbox.EventResize:
				screen.Resize()
			}
		}
	}

	log.Debug("something broke the loop. Time to die")
}

func stream(screen *ui.Screen, stream io.ReadCloser, keyboardQueue chan termbox.Event, closeView chan<- struct{}) {
	defer func() {
		closeView <- struct{}{}
	}()
	screen.Clear()
	screen.Sync()
	v := ui.NewLess()
	go func() {
		stdcopy.StdCopy(v, v, stream)
	}()
	if err := v.Focus(keyboardQueue); err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, err)
	}
	stream.Close()
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()
}

//less shows dry output in a "less" emulator
func less(dry *Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, closeView chan struct{}) {
	defer func() {
		closeView <- struct{}{}
	}()
	screen.Clear()
	v := ui.NewLess()
	v.MarkupSupport()
	go Write(dry, v)
	//Focus blocks until v decides that it does not want focus any more
	if err := v.Focus(keyboardQueue); err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, err)
	}
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()

}
