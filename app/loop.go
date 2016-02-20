package app

import (
	"io"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
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

//RenderLoop renders dry until it quits
func RenderLoop(dry *Dry, screen *ui.Screen) {
	if ok, _ := dry.Ok(); !ok {
		return
	}

	keyboardQueue, done := ui.EventChannel()
	timestampQueue := time.NewTicker(1 * time.Second)

	viewClosed := make(chan struct{}, 1)
	keyboardQueueForView := make(chan termbox.Event)
	dryOutputChan := dry.OuputChannel()
	statusBar := ui.NewStatusBar(0)

	defer timestampQueue.Stop()
	defer close(done)
	defer close(keyboardQueueForView)
	defer close(viewClosed)

	Render(dry, screen, statusBar)
	//focus creation belongs outside the loop
	focus := &focusTracker{&sync.Mutex{}, true}

	go func(focus *focusTracker) {
		for {
			dryMessage, ok := <-dryOutputChan
			if ok {
				if focus.hasFocus() {
					statusBar.StatusMessage(dryMessage, 10*time.Second)
					if dry.Changed() {
						screen.Clear()
						Render(dry, screen, statusBar)
					} else {
						statusBar.Render()
					}
					screen.Flush()
				}
			} else {
				return
			}
		}
	}(focus)

loop:
	for {
		//Used for refresh-forcing events happening outside dry
		var refresh = false
		select {
		case <-timestampQueue.C:
			if focus.hasFocus() {
				timestamp := time.Now().Format(`15:04:05`)
				screen.RenderLine(0, 0, `<right><white>`+timestamp+`</></right>`)
				screen.Flush()
			}
		case <-viewClosed:
			focus.set(true)
			dry.ShowMainView()
			refresh = true
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				if focus.hasFocus() {
					if event.Ch == 'q' || event.Ch == 'Q' {
						break loop
					} else {
						handler := eventHandlerFactory(dry, screen, keyboardQueueForView, viewClosed)
						if handler != nil {
							r, f := handler.handle(event)
							refresh = r
							focus.set(f)
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
				refresh = true
			}
		}
		if focus.hasFocus() && refresh {
			screen.Clear()
			Render(dry, screen, statusBar)
		}
	}

	log.Debug("something broke the loop. Time to die")
}

func stream(screen *ui.Screen, stream io.ReadCloser, keyboardQueue chan termbox.Event, done chan<- struct{}) {
	screen.Clear()
	screen.Sync()
	v := ui.NewLess()
	go func() {
		io.Copy(v, stream)
	}()
	if err := v.Focus(keyboardQueue); err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, err)
	}
	stream.Close()
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()
	done <- struct{}{}
}

//autorefresh view that autorefreshes its content every second
func autorefresh(dry *Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, done chan<- struct{}, doneStats chan<- bool, errC <-chan error) {
	screen.Clear()
	v := ui.NewMarkupView("", 0, 0, screen.Width, screen.Height, false)
	//used to coordinate rendering between the ticker
	//and the exit event
	var mutex = &sync.Mutex{}
	Write(dry, v)
	err := v.Render()
	if err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, err)
	}
	screen.Flush()
	//the ticker is created after the first render
	timestampQueue := time.NewTicker(1000 * time.Millisecond)

loop:
	for {
		select {
		case <-errC:
			{
				mutex.Lock()
				timestampQueue.Stop()
				break loop
			}
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				if event.Key == termbox.KeyEsc {
					//the lock is acquired and the time-based refresh queue is stopped
					//before breaking the loop
					mutex.Lock()
					timestampQueue.Stop()
					break loop
				}
			}
		case <-timestampQueue.C:
			{
				mutex.Lock()
				v.Clear()
				Write(dry, v)
				v.Render()
				screen.Flush()
				mutex.Unlock()
			}
		}
	}
	//cleanup before exiting, the screen is cleared and the lock released
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()
	mutex.Unlock()
	doneStats <- true
	done <- struct{}{}
}

//less shows dry output in a "less" emulator
func less(dry *Dry, screen *ui.Screen, keyboardQueue chan termbox.Event, done chan struct{}) {
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

	done <- struct{}{}
}
