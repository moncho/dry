package app

import (
	"sync"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

var viewsToHandlers = map[viewMode]eventHandler{
	Images:       &imagesScreenEventHandler{},
	Networks:     &networksScreenEventHandler{},
	DiskUsage:    &diskUsageScreenEventHandler{},
	Main:         &containersScreenEventHandler{},
	Monitor:      &monitorScreenEventHandler{},
	Nodes:        &nodesScreenEventHandler{},
	Tasks:        &taskScreenEventHandler{},
	Services:     &servicesScreenEventHandler{},
	ServiceTasks: &serviceTasksScreenEventHandler{},
	Stacks:       &stacksScreenEventHandler{},
	StackTasks:   &stackTasksScreenEventHandler{},
}

var defaultHandler eventHandler

//eventHandler interface to handle termbox events
type eventHandler interface {
	getEventChan() chan termbox.Event
	//handle handles a termbox event
	handle(event termbox.Event)
	//hasFocus returns true while the handler is processing events
	hasFocus() bool
	initialize(dry *Dry,
		screen *ui.Screen,
		keyboardQueueForView chan termbox.Event,
		viewClosedChan chan struct{})
	setForwardEvents(forwardEvents bool)
	widget() appui.AppWidget
	widgetRegistry() *WidgetRegistry
}

type baseEventHandler struct {
	dry              *Dry
	screen           *ui.Screen
	eventChan        chan termbox.Event
	closeViewChan    chan struct{}
	focus            bool
	forwardingEvents bool

	sync.RWMutex
}

func (b *baseEventHandler) getEventChan() chan termbox.Event {
	return b.eventChan
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
	case termbox.KeyF8: // docker events
		dry.ShowDiskUsage()
	case termbox.KeyF9: // docker events
		dry.ShowDockerEvents()
		focus = false
		go appui.Less(renderDry(dry), screen, b.eventChan, b.closeViewChan)
	case termbox.KeyF10: // docker info
		dry.ShowInfo()
		focus = false
		go appui.Less(renderDry(dry), screen, b.eventChan, b.closeViewChan)
	}
	switch event.Ch {
	case '?', 'h', 'H': //help
		focus = false
		dry.ShowHelp()
		go appui.Less(renderDry(dry), screen, b.eventChan, b.closeViewChan)
	case '1':
		cursor.Reset()
		dry.ShowContainers()
	case '2':
		cursor.Reset()
		dry.ShowImages()
	case '3':
		cursor.Reset()
		dry.ShowNetworks()
	case '4':
		cursor.Reset()
		dry.ShowNodes()
	case '5':
		cursor.Reset()
		dry.ShowServices()
	case '6':
		cursor.Reset()
		dry.ShowStacks()
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
		refreshScreen()
	}
}

func (b *baseEventHandler) hasFocus() bool {
	b.RLock()
	defer b.RUnlock()
	return b.focus
}

func (b *baseEventHandler) initialize(dry *Dry,
	screen *ui.Screen,
	keyboardQueueForView chan termbox.Event,
	closeViewChan chan struct{}) {
	b.dry = dry
	b.screen = screen
	b.eventChan = keyboardQueueForView
	b.closeViewChan = closeViewChan
}

func (b *baseEventHandler) setFocus(focus bool) {
	b.Lock()
	defer b.Unlock()
	b.focus = focus
}

func (b *baseEventHandler) setForwardEvents(forward bool) {
	b.forwardingEvents = forward
}

func (b *baseEventHandler) widget() appui.AppWidget {
	return nil
}

func (b *baseEventHandler) widgetRegistry() *WidgetRegistry {
	return b.dry.widgetRegistry
}

type eventHandlerFactory struct {
	dry                  *Dry
	screen               *ui.Screen
	keyboardQueueForView chan termbox.Event
	viewClosed           chan struct{}
	handlers             map[viewMode]eventHandler
	once                 sync.Once
}

//handlerFor creates eventHandlers
func (eh *eventHandlerFactory) handlerFor(view viewMode) eventHandler {

	eh.once.Do(func() {
		defaultHandler = &baseEventHandler{}
		defaultHandler.initialize(eh.dry, eh.screen, eh.keyboardQueueForView, eh.viewClosed)
		eh.handlers = viewsToHandlers
		for _, handler := range eh.handlers {
			handler.initialize(eh.dry, eh.screen, eh.keyboardQueueForView, eh.viewClosed)
		}
	})
	if handler, ok := eh.handlers[view]; ok {
		return handler
	}
	return defaultHandler
}
