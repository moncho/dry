package ui

import "github.com/nsf/termbox-go"

//Focusable define ui elements that can focus
type Focusable interface {
	Focus(events <-chan termbox.Event, done func()) error
}
