package ui

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

//This files adds functions that serve as building blocks for rendering structs
//defined in the ui package.

//fill fills the screen with the given cell starting at x,y until w,h.
func fill(x, y, w, h int, cell termbox.Cell) {
	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			termbox.SetCell(x+lx, y+ly, cell.Ch, cell.Fg, cell.Bg)
		}
	}
}

//renderString renders the given string starting at x, y in the screen
func renderString(x, y int, word string, foreground, background termbox.Attribute) {
	for _, char := range word {
		termbox.SetCell(x, y, char, foreground, background)
		x += runewidth.RuneWidth(char)
	}
}

// renderLineWithMarkup takes the incoming string, uses the given markup to tokenize it and to extract markup
// elements, and displays it all starting at (x,y) location.
func renderLineWithMarkup(x, y, maxWidth int, str string, markup *Markup) {
	start, column := 0, 0

	for _, token := range Tokenize(str, markup.supportedTags()) {
		// First check if it's a tag. Tags are eaten up and not displayed.
		if markup.IsTag(token) {
			continue
		}

		// Here comes the actual text: display it one character at a time.
		for i, char := range token {
			if !markup.RightAligned {
				start = x + column
				column++
			} else {
				start = maxWidth - len(token) + i
			}
			termbox.SetCell(start, y, char, markup.Foreground, markup.Background)
		}
	}
}

func runeAdvanceLen(r rune, pos int) int {
	if r == '\t' {
		return tabstopLength - pos%tabstopLength
	}
	return 1
}

func vOffsetToCOffset(text []byte, boffset int) (voffset, coffset int) {
	text = text[:boffset]
	for len(text) > 0 {
		r, size := utf8.DecodeRune(text)
		text = text[size:]
		coffset++
		voffset += runeAdvanceLen(r, voffset)
	}
	return
}

func byteSliceGrow(s []byte, desiredCap int) []byte {
	if cap(s) < desiredCap {
		ns := make([]byte, len(s), desiredCap)
		copy(ns, s)
		return ns
	}
	return s
}

func byteSliceRemove(text []byte, from, to int) []byte {
	size := to - from
	copy(text[from:], text[to:])
	text = text[:len(text)-size]
	return text
}

func byteSliceInsert(text []byte, offset int, what []byte) []byte {
	n := len(text) + len(what)
	text = byteSliceGrow(text, n)
	text = text[:n]
	copy(text[offset+len(what):], text[offset:])
	copy(text[offset:], what)
	return text
}

// EventChannel returns a channel with termbox's events.
func EventChannel() (<-chan termbox.Event, chan struct{}) {
	// termbox.PollEvent() can get stuck on unexpected signal
	// handling cases, so termbox polling is done is a separate goroutine
	evCh := make(chan termbox.Event)
	done := make(chan struct{})
	go func() {
		defer func() { recover() }()
		defer func() { close(evCh) }()
		for {
			select {
			case evCh <- termbox.PollEvent():
			case <-done:
				return
			}
		}
	}()
	return evCh, done

}
