package app

import (
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
)

var viewsToHandlers map[viewMode]eventHandler

// eventHandler interface to handle terminal events.
type eventHandler interface {
	//handle handles the given event key, the given func can be
	//used to set the handler of the next event
	handle(event *tcell.EventKey, nextHandler func(eventHandler))
}

type baseEventHandler struct {
	dry    *Dry
	screen *ui.Screen
}

func (b *baseEventHandler) handle(event *tcell.EventKey, f func(eventHandler)) {
	dry := b.dry
	screen := b.screen
	cursor := screen.Cursor()
	refresh := true
	switch event.Key() {
	case tcell.KeyUp, tcell.KeyCtrlP: //cursor up
		cursor.ScrollCursorUp()
	case tcell.KeyDown, tcell.KeyCtrlN: // cursor down
		cursor.ScrollCursorDown()
	case tcell.KeyF7: // toggle show header
		dry.toggleShowHeader()
	case tcell.KeyF8: // disk usage
		f(viewsToHandlers[DiskUsage])
		dry.changeView(DiskUsage)
		if du, err := b.dry.dockerDaemon.DiskUsage(); err == nil {
			widgets.DiskUsage.PrepareToRender(&du, nil)
		}
	case tcell.KeyF9: // docker events
		refresh = false
		view := dry.viewMode()
		dry.changeView(EventsMode)
		eh := newEventForwarder()
		f(eh)

		renderer := appui.NewDockerEventsRenderer(dry.dockerDaemon.EventLog().Events())

		go appui.Less(renderer.String(), screen, eh.events(), func() {
			dry.changeView(view)
			f(viewsToHandlers[view])
			refreshScreen()
		})
	case tcell.KeyF10: // docker info
		refresh = false

		view := dry.viewMode()
		dry.changeView(InfoMode)

		info, err := dry.dockerDaemon.Info()
		if err == nil {
			eh := newEventForwarder()
			f(eh)

			renderer := appui.NewDockerInfoRenderer(info)

			go appui.Less(renderer.String(), screen, eh.events(), func() {
				dry.changeView(view)
				f(viewsToHandlers[view])
				refreshScreen()
			})
		} else {
			dry.message(
				fmt.Sprintf(
					"There was an error retrieving Docker information: %s", err.Error()))
		}
	}
	switch event.Rune() {
	case 'k':
		cursor.ScrollCursorUp()
	case 'j':
		cursor.ScrollCursorDown()
	case '?', 'h', 'H': //help
		refresh = false

		view := dry.viewMode()
		eh := newEventForwarder()
		f(eh)
		go appui.Less(help, screen, eh.events(), func() {
			dry.changeView(view)
			f(viewsToHandlers[view])
			refreshScreen()
		})
	case '1':
		cursor.Reset()
		f(viewsToHandlers[Main])
		dry.changeView(Main)
	case '2':
		cursor.Reset()
		f(viewsToHandlers[Images])
		dry.changeView(Images)
	case '3':
		cursor.Reset()
		f(viewsToHandlers[Networks])
		dry.changeView(Networks)
	case '4':
		cursor.Reset()
		f(viewsToHandlers[Volumes])
		dry.changeView(Volumes)
	case '5':
		cursor.Reset()
		f(viewsToHandlers[Nodes])
		dry.changeView(Nodes)
	case '6':
		cursor.Reset()
		f(viewsToHandlers[Services])
		dry.changeView(Services)
	case '7':
		cursor.Reset()
		f(viewsToHandlers[Stacks])
		dry.changeView(Stacks)
	case 'm', 'M': //monitor mode
		cursor.Reset()
		f(viewsToHandlers[Monitor])
		dry.changeView(Monitor)
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
		Volumes: &volumesScreenEventHandler{
			baseEventHandler{
				dry:    dry,
				screen: screen,
			},
			widgets.Volumes,
		},
	}

}

type eventHandlerForwarder interface {
	events() <-chan *tcell.EventKey
	handle(event *tcell.EventKey, nextHandler func(eventHandler))
}

func newEventForwarder() eventHandlerForwarder {
	return &eventHandlerForwarderImpl{
		eventChan: make(chan *tcell.EventKey),
	}
}

type eventHandlerForwarderImpl struct {
	eventChan chan *tcell.EventKey
}

func (b *eventHandlerForwarderImpl) events() <-chan *tcell.EventKey {
	return b.eventChan
}

func (b *eventHandlerForwarderImpl) handle(event *tcell.EventKey, _ func(eventHandler)) {
	b.eventChan <- event
}
