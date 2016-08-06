package ui

import (
	"fmt"

	"github.com/nsf/termbox-go"
)

//ShowErrorMessage renders the given error message using the given screen
//and waits for a termbox event
func ShowErrorMessage(screen *Screen, keyboardQueue <-chan termbox.Event, close chan<- struct{}, err error) {
	termbox.HideCursor()
	screen.Clear()
	screen.RenderLine(0, 0, "There was an error rendering content.")
	screen.RenderLine(0, 1, fmt.Sprintf("Error: %s.", err))
	screen.RenderLine(0, 2, "Press any key to continue.")
	screen.Flush()
	<-keyboardQueue
	close <- struct{}{}
}
