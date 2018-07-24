package app

import (
	"fmt"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

var viewsToHandlers map[viewMode]eventHandler

//eventHandler interface to handle termbox events
type eventHandler interface {
	//handle handles the given termbox event, the given func can be
	//used to set the handler of the next event
	handle(event termbox.Event, nextHandler func(eventHandler))
}

type baseEventHandler struct {
	dry    *Dry
	screen *ui.Screen
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
		if du, err := b.dry.dockerDaemon.DiskUsage(); err == nil {
			widgets.DiskUsage.PrepareToRender(&du, nil)
		}
	case termbox.KeyF9: // docker events
		refresh = false
		view := dry.viewMode()
		dry.SetViewMode(EventsMode)
		eh := newEventForwarder()
		f(eh)

		renderer := appui.NewDockerEventsRenderer(dry.dockerDaemon.EventLog().Events())

		go appui.Less(renderer, screen, eh.events(), func() {
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
			eh := newEventForwarder()
			f(eh)

			renderer := appui.NewDockerInfoRenderer(info)

			go appui.Less(renderer, screen, eh.events(), func() {
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
		eh := newEventForwarder()
		f(eh)
		go appui.Less(ui.StringRenderer(help), screen, eh.events(), func() {
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
				dry:    dry,
				screen: screen,
			},
		},
		Images: &imagesScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widgets.ImageList,
		},
		Networks: &networksScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widgets.Networks,
		},
		DiskUsage: &diskUsageScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
		},
		Main: &containersScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widgets.ContainerList,
		},
		Monitor: &monitorScreenEventHandler{
			baseEventHandler: baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widget: widgets.Monitor,
		},
		Nodes: &nodesScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widgets.Nodes,
		},
		Tasks: &taskScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widgets.NodeTasks,
		},
		Services: &servicesScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widgets.ServiceList,
		},
		ServiceTasks: &serviceTasksScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widgets.ServiceTasks,
		},
		Stacks: &stacksScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widgets.Stacks,
		},
		StackTasks: &stackTasksScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widgets.StackTasks,
		},
	}

}

type eventHandlerForwarder interface {
	events() chan termbox.Event
	handle(event termbox.Event, nextHandler func(eventHandler))
}

func newEventForwarder() eventHandlerForwarder {
	return &eventHandlerForwarderImpl{
		eventChan: make(chan termbox.Event),
	}
}

type eventHandlerForwarderImpl struct {
	eventChan chan termbox.Event
}

func (b *eventHandlerForwarderImpl) events() chan termbox.Event {
	return b.eventChan
}

func (b *eventHandlerForwarderImpl) handle(event termbox.Event, f func(eventHandler)) {
	b.eventChan <- event
}
