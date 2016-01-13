package ui

import (
	"fmt"

	"github.com/nsf/termbox-go"
)

//ShowErrorMessage renders the given error message using the given screen
//and waits for a termbox event
func ShowErrorMessage(screen *Screen, keyboardQueue <-chan termbox.Event, err error) {
	termbox.HideCursor()
	screen.Clear()
	screen.RenderLine(0, 0, fmt.Sprintf("There was an error rendering content. Error: %s.", err))
	screen.RenderLine(0, 1, "Press any key to continue.")
	screen.Flush()
	<-keyboardQueue
}
