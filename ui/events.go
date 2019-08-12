package ui

import "github.com/gdamore/tcell"

//EventSource defines a source of termbox events
type EventSource struct {
	Events               <-chan *tcell.EventKey
	EventHandledCallback func(*tcell.EventKey) error
}

//EventHandlerWidget defines how UI widgets handle termbox events
type EventHandlerWidget interface {
	OnFocus(event EventSource) error
}
