package app

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/appui"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

type imagesScreenEventHandler struct {
	baseEventHandler
	widget *appui.DockerImagesWidget
}

func (h *imagesScreenEventHandler) handle(event termbox.Event, f func(eventHandler)) {
	if h.forwardingEvents() {
		h.eventChan <- event
		return
	}
	handled := h.handleKeyEvent(event.Key, f)

	if !handled {
		handled = h.handleChEvent(event.Ch, f)
	}
	if handled {
		refreshScreen()
	} else {
		h.baseEventHandler.handle(event, f)
	}
}

func (h *imagesScreenEventHandler) handleKeyEvent(key termbox.Key, f func(eventHandler)) bool {
	handled := true
	switch key {
	case termbox.KeyF1: //sort
		h.widget.Sort()
	case termbox.KeyF5: // refresh
		h.widget.Unmount()
	case termbox.KeyCtrlD: //remove dangling images
		prompt := appui.NewPrompt("Do you want to remove dangling images? (y/N)")
		widgets.add(prompt)
		h.setForwardEvents(true)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			h.setForwardEvents(false)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {
				return
			}

			h.dry.appmessage("<red>Removing dangling images</>")
			if count, err := h.dry.dockerDaemon.RemoveDanglingImages(); err == nil {
				h.dry.appmessage(fmt.Sprintf("<red>Removed %d dangling images</>", count))
			} else {
				h.dry.appmessage(
					fmt.Sprintf(
						"<red>Error removing dangling images. %s</>", err))
			}
			refreshScreen()

		}()

	case termbox.KeyCtrlE: //remove image

		prompt := appui.NewPrompt("Do you want to remove the selected image? (y/N)")
		widgets.add(prompt)
		h.setForwardEvents(true)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			h.setForwardEvents(false)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {
				return
			}

			rmImage := func(id string) error {
				shortID := drydocker.TruncateID(id)
				if _, err := h.dry.dockerDaemon.Rmi(id, false); err == nil {
					h.dry.appmessage(fmt.Sprintf("<red>Removed image:</> <white>%s</>", shortID))
				} else {
					h.dry.appmessage(fmt.Sprintf("<red>Error removing image </><white>%s: %s</>", shortID, err.Error()))
				}
				return nil
			}
			if err := h.widget.OnEvent(rmImage); err != nil {
				h.dry.appmessage(
					fmt.Sprintf("Error removing image: %s", err.Error()))
			}
			refreshScreen()

		}()

	case termbox.KeyCtrlF: //force remove image
		prompt := appui.NewPrompt("Do you want to remove the selected image? (y/N)")
		widgets.add(prompt)
		h.setForwardEvents(true)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: h.eventChan,
				EventHandledCallback: func(e termbox.Event) error {
					return refreshScreen()
				},
			}
			prompt.OnFocus(events)
			conf, cancel := prompt.Text()
			h.setForwardEvents(false)
			widgets.remove(prompt)
			if cancel || (conf != "y" && conf != "Y") {
				return
			}

			rmImage := func(id string) error {
				shortID := drydocker.TruncateID(id)
				if _, err := h.dry.dockerDaemon.Rmi(id, true); err == nil {
					h.dry.appmessage(fmt.Sprintf("<red>Removed image:</> <white>%s</>", shortID))
				} else {
					h.dry.appmessage(fmt.Sprintf("<red>Error removing image </><white>%s: %s</>", shortID, err.Error()))
				}
				return nil
			}
			if err := h.widget.OnEvent(rmImage); err != nil {
				h.dry.appmessage(
					fmt.Sprintf("Error forcing image removal: %s", err.Error()))
			}
			refreshScreen()

		}()

	case termbox.KeyEnter: //inspect image
		h.setForwardEvents(true)
		inspectImage := inspect(
			h.screen,
			h.eventChan,
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.InspectImage(id)
			},
			func() {
				h.setForwardEvents(false)
				h.dry.SetViewMode(Images)
				f(h)
				refreshScreen()
			})

		if err := h.widget.OnEvent(inspectImage); err != nil {
			h.dry.appmessage(
				fmt.Sprintf("Error inspecting image: %s", err.Error()))
		}

	default:
		handled = false
	}
	return handled
}

func (h *imagesScreenEventHandler) handleChEvent(ch rune, f func(eventHandler)) bool {
	dry := h.dry
	handled := true
	switch ch {
	case '2': //Ignore since dry is already on the images screen

	case 'i', 'I': //image history

		showHistory := func(id string) error {
			history, err := dry.dockerDaemon.History(id)

			if err == nil {
				h.setForwardEvents(true)
				renderer := appui.NewDockerImageHistoryRenderer(history)

				go appui.Less(renderer, h.screen, h.eventChan, func() {
					h.dry.SetViewMode(Images)
					f(h)
					h.setForwardEvents(false)
					refreshScreen()
				})
			}
			return err
		}
		if err := h.widget.OnEvent(showHistory); err != nil {
			dry.appmessage(err.Error())
		}
	case 'r', 'R': //Run container
		runImage := func(id string) error {
			h.setForwardEvents(true)
			defer h.setForwardEvents(false)
			image, err := h.dry.dockerDaemon.ImageByID(id)
			if err != nil {
				return err
			}
			rw := appui.NewImageRunWidget(image)
			widgets.add(rw)
			go func(image types.ImageSummary) {
				events := ui.EventSource{
					Events: h.eventChan,
					EventHandledCallback: func(e termbox.Event) error {
						return refreshScreen()
					},
				}
				rw.OnFocus(events)
				widgets.remove(rw)
				runCommand, canceled := rw.Text()
				if canceled {
					return
				}
				if err := dry.dockerDaemon.RunImage(image, runCommand); err != nil {
					dry.appmessage(err.Error())
				}

			}(image)
			return nil
		}
		if err := h.widget.OnEvent(runImage); err != nil {
			dry.appmessage(
				fmt.Sprintf("Error running image: %s", err.Error()))
		}
	case '%':
		h.setForwardEvents(true)
		applyFilter := func(filter string, canceled bool) {
			if !canceled {
				h.widget.Filter(filter)
			}
			h.setForwardEvents(false)
		}
		showFilterInput(newEventSource(h.eventChan), applyFilter)
	default:
		handled = false
	}
	return handled
}
