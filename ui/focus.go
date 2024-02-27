package ui

import "github.com/gdamore/tcell"

// Focusable define ui elements that can focus
type Focusable interface {
	Focus(events <-chan *tcell.EventKey, done func()) error
}
