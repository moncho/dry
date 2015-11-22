package ui

import tb "github.com/nsf/termbox-go"

type Key struct {
	KeyCodes []rune
	Keys     []tb.Key
	HelpText string
}
