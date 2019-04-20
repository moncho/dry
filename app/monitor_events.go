package app

import (
	"errors"
	"fmt"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
	"strconv"
)

type monitorScreenEventHandler struct {
	baseEventHandler
	widget *appui.Monitor
}

func (h *monitorScreenEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	handled := false
	cursor := h.screen.Cursor
	switch event.Key {
	case termbox.KeyF1:
		handled = true
		h.widget.Sort()
		h.widget.OnEvent(nil)
	case termbox.KeyArrowUp: //cursor up
		handled = true
		cursor.ScrollCursorUp()
		h.widget.OnEvent(nil)
	case termbox.KeyArrowDown: // cursor down
		handled = true
		cursor.ScrollCursorDown()
		h.widget.OnEvent(nil)
	case termbox.KeyEnter: //Container menu
		showMenu := func(id string) error {
			h.widget.Unmount()
			h.screen.Cursor.Reset()
			widgets.ContainerMenu.ForContainer(id)
			widgets.ContainerMenu.OnUnmount = func() error {
				h.screen.Cursor.Reset()
				h.dry.changeView(Monitor)
				f(h)
				return refreshScreen()
			}
			h.dry.changeView(ContainerMenu)
			f(viewsToHandlers[ContainerMenu])
			return refreshScreen()
		}
		if err := h.widget.OnEvent(showMenu); err != nil {
			h.dry.message(err.Error())
		}
	}
	if !handled {
		switch event.Ch {
		case 'g': //Cursor to the top
			handled = true
			cursor.Reset()
			h.widget.OnEvent(nil)

		case 'G': //Cursor to the bottom
			handled = true
			cursor.Bottom()
			h.widget.OnEvent(nil)
		case 's': // Set the delay between updates to <delay> seconds.
			//widget is mounted on render, dont Mount here
			h.widget.Unmount()
			prompt := appui.NewPrompt("Set the delay between updates (in milliseconds)")
			widgets.add(prompt)
			forwarder := newEventForwarder()
			f(forwarder)
			h.dry.changeView(NoView)
			refreshScreen()
			go func() {
				defer h.dry.changeView(Monitor)
				defer f(h)
				events := ui.EventSource{
					Events: forwarder.events(),
					EventHandledCallback: func(e termbox.Event) error {
						return refreshScreen()
					},
				}
				prompt.OnFocus(events)
				input, cancel := prompt.Text()
				widgets.remove(prompt)
				if cancel {
					return
				}
				refreshRate, err := toInt(input)
				if err != nil {
					h.dry.message(
						fmt.Sprintf("Error setting refresh rate: %s", err.Error()))
					return
				}
				h.widget.RefreshRate(refreshRate)
			}()
		}
	}
	if !handled {
		nh := func(eh eventHandler) {
			if cancelMonitorWidget != nil {
				cancelMonitorWidget()
				cancelMonitorWidget = nil
			}
			f(eh)
		}
		h.baseEventHandler.handle(event, nh)
	}
}

func toInt(s string) (int, error) {
	i, err := strconv.Atoi(s)

	if err != nil {
		return -1, errors.New("Be nice, a number is expected")
	}
	if i < 0 {
		return -1, errors.New("Negative values are not allowed")
	}
	return i, nil
}
