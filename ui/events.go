package ui

import termbox "github.com/nsf/termbox-go"

//EventSource defines a source of termbox events
type EventSource struct {
	Events               <-chan termbox.Event
	EventHandledCallback func(termbox.Event) error
}

//EventHandlerWidget defines how UI widgets handle termbox events
type EventHandlerWidget interface {
	OnFocus(event EventSource) error
}
