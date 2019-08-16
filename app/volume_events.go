package app

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
)

type volumesScreenEventHandler struct {
	baseEventHandler
	widget *appui.VolumesWidget
}

func (h *volumesScreenEventHandler) handle(event *tcell.EventKey, f func(eh eventHandler)) {
	dry := h.dry
	screen := h.screen
	handled := true
	switch event.Key() {
	case tcell.KeyF1: //sort
		h.widget.Sort()
		refreshScreen()
	case tcell.KeyF5: // refresh
		h.dry.message("Refreshing volumes list")
		h.widget.Unmount()
		refreshScreen()
	case tcell.KeyEnter: //inspect
		forwarder := newEventForwarder()
		f(forwarder)
		inspect := inspect(screen, forwarder.events(),
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.VolumeInspect(context.Background(), id)
			},
			func() {
				h.dry.changeView(Volumes)
				f(h)
				refreshScreen()
			})

		if err := h.widget.OnEvent(inspect); err != nil {
			dry.message(
				fmt.Sprintf("Error inspecting volume: %s", err.Error()))
		}
	case tcell.KeyCtrlA: //remove all

		prompt := appui.NewPrompt("Do you want to remove all volumes? (y/N)")
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			f(h)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {
				return
			}

			h.dry.message("<red>Removing all volumes</>")
			if count, err := h.dry.dockerDaemon.VolumeRemoveAll(context.Background()); err == nil {
				h.dry.message(fmt.Sprintf("<red>Removed %d volumes</>", count))
			} else {
				h.dry.message(
					fmt.Sprintf(
						"<red>Error removing volumes: %s</>", err))
			}
			refreshScreen()

		}()

	case tcell.KeyCtrlE: //remove volume

		prompt := appui.NewPrompt("Do you want to remove the selected volume? (y/N)")
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			f(h)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {
				return
			}

			rmVolume := func(id string) error {

				if err := h.dry.dockerDaemon.VolumeRemove(context.Background(), id, false); err == nil {
					h.dry.message(fmt.Sprintf("<red>Removed volume:</> <white>%s</>", id))
				} else {
					h.dry.message(fmt.Sprintf("<red>Error removing volume </><white>%s: %s</>", id, err.Error()))
				}

				return nil
			}
			if err := h.widget.OnEvent(rmVolume); err != nil {
				dry.message(
					fmt.Sprintf("Error removing volume: %s", err.Error()))
			}
			refreshScreen()

		}()
	case tcell.KeyCtrlF: //force volume removal

		prompt := appui.NewPrompt("Do you want to remove the selected volume? (y/N)")
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			f(h)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {
				return
			}

			rmVolume := func(id string) error {

				if err := h.dry.dockerDaemon.VolumeRemove(context.Background(), id, true); err == nil {
					h.dry.message(fmt.Sprintf("<red>Removed volume:</> <white>%s</>", id))
				} else {
					h.dry.message(fmt.Sprintf("<red>Error removing volume </><white>%s: %s</>", id, err.Error()))
				}

				return nil
			}
			if err := h.widget.OnEvent(rmVolume); err != nil {
				dry.message(
					fmt.Sprintf("Error removing volume: %s", err.Error()))
			}
			refreshScreen()

		}()
	case tcell.KeyCtrlU: //remove unused volumes
		prompt := appui.NewPrompt("Do you want to remove unused volumes? (y/N)")
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e *tcell.EventKey) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			f(h)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {
				return
			}

			h.dry.message("<red>Removing unused volumes</>")
			if count, err := h.dry.dockerDaemon.VolumePrune(context.Background()); err == nil {
				h.dry.message(fmt.Sprintf("<red>Removed %d unused volumes</>", count))
			} else {
				h.dry.message(
					fmt.Sprintf(
						"<red>Error removing unused volumes: %s</>", err))
			}
			refreshScreen()

		}()
	default:
		handled = false
	}
	if !handled {
		switch event.Rune() {
		case '4':
			//already in volumes screen
			handled = true
		case '%':
			handled = true
			forwarder := newEventForwarder()
			f(forwarder)
			refreshScreen()
			applyFilter := func(filter string, canceled bool) {
				if !canceled {
					h.widget.Filter(filter)
				}
				f(h)
			}
			showFilterInput(newEventSource(forwarder.events()), applyFilter)
		}
	}
	if !handled {
		h.baseEventHandler.handle(event, f)
	}
}
