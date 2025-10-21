package app

import (
	"strings"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
)

func logsPrompt() *appui.Prompt {
	return appui.NewPrompt("Show logs since timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes) or leave empty")
}

func newEventSource(events <-chan *tcell.EventKey) ui.EventSource {
	return ui.EventSource{
		Events: events,
		EventHandledCallback: func(e *tcell.EventKey) error {
			return refreshScreen()
		},
	}
}

func inspect(
	screen *ui.Screen,
	events <-chan *tcell.EventKey,
	inspect func(id string) (any, error),
	onClose func()) func(id string) error {
	return func(id string) error {
		inspected, err := inspect(id)
		if err != nil {
			return err
		}
		renderer := appui.NewJSONRenderer(inspected)
		go appui.Less(renderer.String(), screen, events, onClose)
		return nil
	}
}

func curateLogsDuration(s string) string {
	neg := strings.Index(s, "-")
	if neg >= 0 {
		return s[neg+1:]
	}
	return s

}
