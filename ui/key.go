package ui

import "github.com/gdamore/tcell"

// Key represents a keyboard key
// Not in use.
type Key struct {
	KeyCodes []rune
	Keys     []tcell.Key
	HelpText string
}
