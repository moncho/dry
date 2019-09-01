package ui

import "github.com/gdamore/tcell"

// EventSource defines a source of keypress events.
type EventSource struct {
	Events               <-chan *tcell.EventKey
	EventHandledCallback func(*tcell.EventKey) error
}
