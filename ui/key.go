package ui

import tb "github.com/nsf/termbox-go"

//Key represents a keyboard key
//Not in use.
type Key struct {
	KeyCodes []rune
	Keys     []tb.Key
	HelpText string
}
