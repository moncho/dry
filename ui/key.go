package ui

import tb "github.com/nsf/termbox-go"

const (
	//Magic number to represent Alt+e key char. There is probably a better way
	KeyAlte = 8364
)

//Key represents a keyboard key
//Not in use.
type Key struct {
	KeyCodes []rune
	Keys     []tb.Key
	HelpText string
}
