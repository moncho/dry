package ui

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

//Functions that serve as building blocks for rendering structs
//defined in the ui package.

//fill fills the screen with the given cell starting at x,y until w,h.
func fill(x, y, w, h int, cell termbox.Cell) {
	for lx := 0; lx < w; lx++ {
		termbox.SetCell(x+lx, y, cell.Ch, cell.Fg, cell.Bg)
	}
}

//renderString renders the given string starting at x, y in the screen, returns the
//rune-width of the given string and the number of screen lines used to render
func renderString(x, y, maxWidth int, s string, foreground, background termbox.Attribute) (int, int) {
	stringWidth := 0
	virtualScreenWidth := maxWidth
	//tracks the number of screen lines used to render
	additionalLines := 0
	startCol := x

	for _, char := range s {
		runewidth := runewidth.RuneWidth(char)
		stringWidth += runewidth
		//Check if a new line is going to be needed
		if stringWidth > virtualScreenWidth {
			//A new line is going to be used, the virtual screen width has to be
			//extended
			virtualScreenWidth += virtualScreenWidth + maxWidth
			additionalLines++
			y += additionalLines
			//new line, start column goes back to the beginning
			startCol = x
		}
		termbox.SetCell(startCol, y, char, foreground, background)
		startCol += runewidth

	}
	return stringWidth, additionalLines + 1
}

// renderLineWithMarkup takes the incoming string, uses the given markup to tokenize it and to extract markup
// elements, and displays it all starting at (x,y) location.
// returns the number of screen lines used to render the line
func renderLineWithMarkup(x, y, maxWidth int, str string, markup *Markup) int {
	start, column := 0, 0

	stringWidth := 0
	//tracks the number of screen lines used to render
	additionalLines := 0

	availableWidth := maxWidth

	for _, token := range Tokenize(str, SupportedTags) {
		// First check if it's a tag. Tags are eaten up and not displayed.
		if markup.IsTag(token) {
			continue
		}

		// Here comes the actual text: display it one character at a time.
		for i, char := range token {
			runewidth := runewidth.RuneWidth(char)
			stringWidth += runewidth
			//Check if a new line is going to be needed
			if stringWidth > availableWidth {
				//A new line is going to be used, the screen width has doubled
				availableWidth = availableWidth * 2
				additionalLines++
				y += additionalLines
				start = 0

			}
			if !markup.RightAligned {
				start = x + column
				column++
			} else {
				start = maxWidth - len(token) + i
			}
			termbox.SetCell(start, y, char, markup.Foreground, markup.Background)
		}
	}
	//At least one screen line has been used
	return additionalLines + 1
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

// EventChannel returns a channel with termbox's events and a done channel.
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
