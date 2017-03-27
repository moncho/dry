package app

import (
	"sync"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//eventHandler interface to handle termbox events
type eventHandler interface {
	//handle handles a termbox event
	handle(event termbox.Event)
	//hasFocus returns true while the handler is processing events
	hasFocus() bool
	initialize(dry *Dry,
		screen *ui.Screen,
		keyboardQueueForView chan termbox.Event,
		viewClosedChan chan struct{},
		renderChan chan<- struct{})
}

type baseEventHandler struct {
	dry                  *Dry
	screen               *ui.Screen
	keyboardQueueForView chan termbox.Event
	closeViewChan        chan struct{}
	renderChan           chan<- struct{}
	focus                bool
	sync.RWMutex
}

func (b *baseEventHandler) initialize(dry *Dry,
	screen *ui.Screen,
	keyboardQueueForView chan termbox.Event,
	closeViewChan chan struct{},
	renderChan chan<- struct{}) {
	b.dry = dry
	b.screen = screen
	b.keyboardQueueForView = keyboardQueueForView
	b.closeViewChan = closeViewChan
	b.renderChan = renderChan
}

func (b *baseEventHandler) hasFocus() bool {
	b.RLock()
	defer b.RUnlock()
	return b.focus
}

func (b *baseEventHandler) setFocus(focus bool) {
	b.Lock()
	defer b.Unlock()
	b.focus = focus
}

func (b *baseEventHandler) handle(event termbox.Event) {
	dry := b.dry
	screen := b.screen
	cursor := screen.Cursor
	focus := true
	switch event.Key {
	case termbox.KeyArrowUp: //cursor up
		cursor.ScrollCursorUp()
	case termbox.KeyArrowDown: // cursor down
		cursor.ScrollCursorDown()

	case termbox.KeyF5: // refresh
		dry.Refresh()
	case termbox.KeyF8: // docker events
		dry.ShowDiskUsage()
	case termbox.KeyF9: // docker events
		dry.ShowDockerEvents()
		focus = false
		go appui.Less(renderDry(dry), screen, b.keyboardQueueForView, b.closeViewChan)
	case termbox.KeyF10: // docker info
		dry.ShowInfo()
		focus = false
		go appui.Less(renderDry(dry), screen, b.keyboardQueueForView, b.closeViewChan)
	}
	switch event.Ch {
	case '?', 'h', 'H': //help
		focus = false
		dry.ShowHelp()
		go appui.Less(renderDry(dry), screen, b.keyboardQueueForView, b.closeViewChan)
	case '1':
		cursor.Reset()
		dry.ShowContainers()
	case '2':
		cursor.Reset()
		dry.ShowImages()
	case '3':
		cursor.Reset()
		dry.ShowNetworks()
	case 'm', 'M': //monitor mode
		cursor.Reset()
		dry.ShowMonitor()
	case 'g': //Cursor to the top
		cursor.Reset()
	case 'G': //Cursor to the bottom
		cursor.Bottom()
	}

	b.setFocus(focus)
	if b.hasFocus() {
		b.renderChan <- struct{}{}
	}
}

type eventHandlerFactory struct {
	dry                  *Dry
	screen               *ui.Screen
	keyboardQueueForView chan termbox.Event
	viewClosed           chan struct{}
	renderChan           chan<- struct{}
	handlers             map[viewMode]eventHandler
	once                 sync.Once
}

//handlerFor creates eventHandlers
func (eh *eventHandlerFactory) handlerFor(view viewMode) eventHandler {

	eh.once.Do(func() {
		eh.handlers = make(map[viewMode]eventHandler)
		handler := &imagesScreenEventHandler{}
		handler.initialize(eh.dry, eh.screen, eh.keyboardQueueForView, eh.viewClosed, eh.renderChan)
		eh.handlers[Images] = handler

		iHandler := &networksScreenEventHandler{}
		iHandler.initialize(eh.dry, eh.screen, eh.keyboardQueueForView, eh.viewClosed, eh.renderChan)

		eh.handlers[Networks] = iHandler

		duHandler := &diskUsageScreenEventHandler{}
		duHandler.initialize(eh.dry, eh.screen, eh.keyboardQueueForView, eh.viewClosed, eh.renderChan)

		eh.handlers[DiskUsage] = duHandler

		cHandler := &containersScreenEventHandler{}
		cHandler.initialize(eh.dry, eh.screen, eh.keyboardQueueForView, eh.viewClosed, eh.renderChan)
		eh.handlers[Main] = cHandler

		mHandler := &monitorScreenEventHandler{}
		mHandler.initialize(eh.dry, eh.screen, eh.keyboardQueueForView, eh.viewClosed, eh.renderChan)
		eh.handlers[Monitor] = mHandler

	})

	return eh.handlers[view]
}
