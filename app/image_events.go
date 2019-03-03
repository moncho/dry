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
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e termbox.Event) error {
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

			h.dry.appmessage("<red>Removing dangling images</>")
			if count, err := h.dry.dockerDaemon.RemoveDanglingImages(); err == nil {
				h.dry.appmessage(fmt.Sprintf("<red>Removed %d dangling images</>", count))
			} else {
				h.dry.appmessage(
					fmt.Sprintf(
						"<red>Error removing dangling images: %s</>", err))
			}
			refreshScreen()

		}()

	case termbox.KeyCtrlE: //remove image

		prompt := appui.NewPrompt("Do you want to remove the selected image? (y/N)")
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e termbox.Event) error {
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
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e termbox.Event) error {
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

	case termbox.KeyCtrlU: //remove unused images
		prompt := appui.NewPrompt("Do you want to remove all unused images? (y/N)")
		widgets.add(prompt)
		forwarder := newEventForwarder()
		f(forwarder)
		refreshScreen()
		go func() {
			events := ui.EventSource{
				Events: forwarder.events(),
				EventHandledCallback: func(e termbox.Event) error {
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

			h.dry.appmessage("<red>Removing unused images</>")
			if count, err := h.dry.dockerDaemon.RemoveUnusedImages(); err == nil {
				h.dry.appmessage(fmt.Sprintf("<red>Removed %d images</>", count))
			} else {
				h.dry.appmessage(
					fmt.Sprintf(
						"<red>Error removing unused images: %s</>", err))
			}
			refreshScreen()

		}()

	case termbox.KeyEnter: //inspect image
		forwarder := newEventForwarder()
		f(forwarder)
		inspectImage := inspect(
			h.screen,
			forwarder.events(),
			func(id string) (interface{}, error) {
				return h.dry.dockerDaemon.InspectImage(id)
			},
			func() {
				h.dry.ViewMode(Images)
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
				forwarder := newEventForwarder()
				f(forwarder)
				renderer := appui.NewDockerImageHistoryRenderer(history)

				go appui.Less(renderer, h.screen, forwarder.events(), func() {
					h.dry.ViewMode(Images)
					f(h)
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
			image, err := h.dry.dockerDaemon.ImageByID(id)
			if err != nil {
				return err
			}
			rw := appui.NewImageRunWidget(image)
			widgets.add(rw)
			forwarder := newEventForwarder()
			f(forwarder)
			refreshScreen()
			go func(image types.ImageSummary) {
				defer f(h)
				events := ui.EventSource{
					Events: forwarder.events(),
					EventHandledCallback: func(e termbox.Event) error {
						return refreshScreen()
					},
				}
				rw.OnFocus(events)
				widgets.remove(rw)
				f(h)
				runCommand, canceled := rw.Text()
				if canceled {
					return
				}
				if err := dry.dockerDaemon.RunImage(image, runCommand); err != nil {
					dry.appmessage(err.Error())
				} else {
					var repo string
					if len(image.RepoTags) > 0 {
						repo = image.RepoTags[0]
					}
					dry.appmessage(
						fmt.Sprintf(
							"Image %s run successfully", repo))
				}
				refreshScreen()

			}(image)
			return nil
		}
		if err := h.widget.OnEvent(runImage); err != nil {
			dry.appmessage(
				fmt.Sprintf("Error running image: %s", err.Error()))
		}
	case '%':
		forwarder := newEventForwarder()
		f(forwarder)
		applyFilter := func(filter string, canceled bool) {
			if !canceled {
				h.widget.Filter(filter)
			}
			f(h)
		}
		showFilterInput(newEventSource(forwarder.events()), applyFilter)
	default:
		handled = false
	}
	return handled
}
