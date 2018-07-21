package app

import (
	"fmt"
	"sync"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

var viewsToHandlers = map[viewMode]eventHandler{
	ContainerMenu: &cMenuEventHandler{},
	Images:        &imagesScreenEventHandler{},
	Networks:      &networksScreenEventHandler{},
	DiskUsage:     &diskUsageScreenEventHandler{},
	Main:          &containersScreenEventHandler{},
	Monitor:       &monitorScreenEventHandler{},
	Nodes:         &nodesScreenEventHandler{},
	Tasks:         &taskScreenEventHandler{},
	Services:      &servicesScreenEventHandler{},
	ServiceTasks:  &serviceTasksScreenEventHandler{},
	Stacks:        &stacksScreenEventHandler{},
	StackTasks:    &stackTasksScreenEventHandler{},
}

//eventHandler interface to handle termbox events
type eventHandler interface {
	events() chan termbox.Event
	//handle handles the given termbox event, the given func can be
	//used to set the handler of the next event
	handle(event termbox.Event, nextHandler func(eventHandler))
}

type baseEventHandler struct {
	dry        *Dry
	screen     *ui.Screen
	eventChan  chan termbox.Event
	forwarding bool
	sync.RWMutex
}

func (b *baseEventHandler) events() chan termbox.Event {
	return b.eventChan
}

func (b *baseEventHandler) forwardingEvents() bool {
	b.RLock()
	defer b.RUnlock()
	return b.forwarding
}

func (b *baseEventHandler) setForwardEvents(t bool) {
	b.Lock()
	defer b.Unlock()
	b.forwarding = t
}

func (b *baseEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	dry := b.dry
	screen := b.screen
	cursor := screen.Cursor
	refresh := true
	switch event.Key {
	case termbox.KeyArrowUp: //cursor up
		cursor.ScrollCursorUp()
	case termbox.KeyArrowDown: // cursor down
		cursor.ScrollCursorDown()
	case termbox.KeyF8: // disk usage
		f(viewsToHandlers[DiskUsage])
		dry.SetViewMode(DiskUsage)
	case termbox.KeyF9: // docker events
		refresh = false
		view := dry.viewMode()
		dry.SetViewMode(EventsMode)
		eh := &eventHandlerForwarder{
			eventChan: make(chan termbox.Event),
		}
		f(eh)

		renderer := appui.NewDockerEventsRenderer(dry.dockerDaemon.EventLog().Events())

		go appui.Less(renderer, screen, eh.eventChan, func() {
			dry.SetViewMode(view)
			f(viewsToHandlers[view])
			refreshScreen()
		})
	case termbox.KeyF10: // docker info
		refresh = false

		view := dry.viewMode()
		dry.SetViewMode(InfoMode)

		info, err := dry.dockerDaemon.Info()
		if err == nil {
			eh := &eventHandlerForwarder{
				eventChan: make(chan termbox.Event),
			}
			f(eh)

			renderer := appui.NewDockerInfoRenderer(info)

			go appui.Less(renderer, screen, eh.eventChan, func() {
				dry.SetViewMode(view)
				f(viewsToHandlers[view])
				refreshScreen()
			})
		} else {
			dry.appmessage(
				fmt.Sprintf(
					"There was an error retrieving Docker information: %s", err.Error()))
		}
	}
	switch event.Ch {
	case '?', 'h', 'H': //help
		refresh = false

		view := dry.viewMode()
		eh := &eventHandlerForwarder{
			eventChan: make(chan termbox.Event),
		}
		f(eh)
		go appui.Less(ui.StringRenderer(help), screen, eh.eventChan, func() {
			dry.SetViewMode(view)
			f(viewsToHandlers[view])
			refreshScreen()
		})
	case '1':
		cursor.Reset()
		f(viewsToHandlers[Main])
		dry.SetViewMode(Main)
	case '2':
		cursor.Reset()
		f(viewsToHandlers[Images])
		dry.SetViewMode(Images)
	case '3':
		cursor.Reset()
		f(viewsToHandlers[Networks])
		dry.SetViewMode(Networks)
	case '4':
		cursor.Reset()
		f(viewsToHandlers[Nodes])
		dry.SetViewMode(Nodes)
	case '5':
		cursor.Reset()
		f(viewsToHandlers[Services])
		dry.SetViewMode(Services)
	case '6':
		cursor.Reset()
		f(viewsToHandlers[Stacks])
		dry.SetViewMode(Stacks)
	case 'm', 'M': //monitor mode
		cursor.Reset()
		f(viewsToHandlers[Monitor])
		dry.SetViewMode(Monitor)
	case 'g': //Cursor to the top
		cursor.Reset()
	case 'G': //Cursor to the bottom
		cursor.Bottom()
	}
	if refresh {
		refreshScreen()
	}

}

func initHandlers(dry *Dry, screen *ui.Screen) map[viewMode]eventHandler {
	return map[viewMode]eventHandler{
		ContainerMenu: &cMenuEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
		},
		Images: &imagesScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
			widgets.ImageList,
		},
		Networks: &networksScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
			widgets.Networks,
		},
		DiskUsage: &diskUsageScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
		},
		Main: &containersScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
			widgets.ContainerList,
		},
		Monitor: &monitorScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
			widgets.Monitor,
		},
		Nodes: &nodesScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
			widgets.Nodes,
		},
		Tasks: &taskScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
			widgets.NodeTasks,
		},
		Services: &servicesScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
			widgets.ServiceList,
		},
		ServiceTasks: &serviceTasksScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
			widgets.ServiceTasks,
		},
		Stacks: &stacksScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
			widgets.Stacks,
		},
		StackTasks: &stackTasksScreenEventHandler{
			baseEventHandler{
				dry:       dry,
				screen:    screen,
				eventChan: make(chan termbox.Event),
			},
			widgets.StackTasks,
		},
	}

}

type eventHandlerForwarder struct {
	eventChan chan termbox.Event
}

func (b *eventHandlerForwarder) events() chan termbox.Event {
	return b.eventChan
}

func (b *eventHandlerForwarder) handle(event termbox.Event, f func(eventHandler)) {
	b.eventChan <- event
}
