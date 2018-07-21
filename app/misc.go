package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

func logsPrompt() *appui.Prompt {
	return appui.NewPrompt("Show logs since timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes) or leave empty")
}

func newEventSource(events <-chan termbox.Event) ui.EventSource {
	return ui.EventSource{
		Events: events,
		EventHandledCallback: func(e termbox.Event) error {
			return refreshScreen()
		},
	}
}

func inspect(
	screen *ui.Screen,
	events <-chan termbox.Event,
	inspect func(id string) (interface{}, error),
	onClose func()) func(id string) error {
	return func(id string) error {
		inspected, err := inspect(id)
		if err != nil {
			return err
		}
		renderer := appui.NewJSONRenderer(inspected)
		go appui.Less(renderer, screen, events, onClose)
		return nil
	}
}
