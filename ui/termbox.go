package ui

import (
	"unicode/utf8"

	"github.com/gdamore/tcell"

	"github.com/mattn/go-runewidth"
)

//Functions that serve as building blocks for rendering structs
//defined in the ui package.

// renderLineWithMarkup renders the given string, using the given markup processor to
// identify and ignore markup elements, at the given location.
// returns the number of screen lines used to render the line
func renderLineWithMarkup(x, y, maxWidth int, str string, markup *Markup) int {
	column := x
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
		for _, char := range token {
			runewidth := runewidth.RuneWidth(char)
			stringWidth += runewidth
			//Check if a new line is going to be needed
			if stringWidth > availableWidth {
				//A new line is going to be used, the screen width has doubled
				availableWidth = availableWidth * 2
				additionalLines++
				y += additionalLines
			}
			style := mkStyle(markup.Foreground, markup.Background)
			ActiveScreen.screen.SetCell(column, y, style, char)
			column++
		}
	}
	//At least one screen line has been used
	return additionalLines + 1
}

func runeAdvanceLen(r rune, pos int) int {
	if r == '\t' {
		return tabStopLength - pos%tabStopLength
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

// EventChannel returns a channel on which termbox's events are published.
func EventChannel() (<-chan tcell.Event, chan struct{}) {
	events := make(chan tcell.Event)
	done := make(chan struct{})
	go func() {
		defer func() { close(events) }()

		for {
			events <- ActiveScreen.screen.PollEvent()
			select {
			case <-done:
				return
			default:
			}
		}

	}()
	return events, done
}
