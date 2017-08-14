package ui

import termbox "github.com/nsf/termbox-go"

//EventSource defines a source of termbox events
type EventSource struct {
	Events               <-chan termbox.Event
	EventHandledCallback func() error
}

//EventHandlerWidget defines how UI widget handle termbox events
type EventHandlerWidget interface {
	OnFocus(event EventSource) error
}
