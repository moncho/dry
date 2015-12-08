package ui

import (
	`strings`
	`time`

	"github.com/nsf/termbox-go"
)

// Screen is thin wrapper aroung Termbox library to provide basic display
// capabilities as required by dry.
type Screen struct {
	Width    int     // Current number of columns.
	Height   int     // Current number of rows.
	cleared  bool    // True after the screens gets cleared.
	markup   *Markup // Pointer to markup processor (gets created by screen).
	pausedAt *time.Time
	Cursor   *Cursor // Pointer to cursor (gets created by screen).
}

type Cursor struct {
	Line int
	Fg   termbox.Attribute
	Ch   rune
}

// Initializes Termbox, creates screen along with layout and markup, and
// calculates current screen dimensions. Once initialized the screen is
// ready for display.
func NewScreen() *Screen {

	if err := termbox.Init(); err != nil {
		panic(err)
	}
	termbox.SetOutputMode(termbox.Output256)
	screen := &Screen{}
	screen.markup = NewMarkup()
	screen.Cursor = &Cursor{Line: 0, Fg: termbox.ColorRed, Ch: '옷'}

	return screen.Resize()
}

// Close gets called upon program termination to close the Termbox.
func (screen *Screen) Close() *Screen {
	termbox.Close()
	return screen
}

// Resize gets called when the screen is being resized. It recalculates screen
// dimensions and requests to clear the screen on next update.
func (screen *Screen) Resize() *Screen {
	screen.Width, screen.Height = termbox.Size()
	screen.cleared = false
	return screen
}

// Pause is a toggle function that either creates a timestamp of the pause
// request or resets it to nil.
func (screen *Screen) Pause(pause bool) *Screen {
	if pause {
		screen.pausedAt = new(time.Time)
		*screen.pausedAt = time.Now()
	} else {
		screen.pausedAt = nil
	}

	return screen
}

// Clear makes the entire screen blank using default background color.
func (screen *Screen) Clear() *Screen {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	screen.cleared = true
	termbox.Flush()
	return screen
}

func (screen *Screen) Sync() *Screen {
	termbox.Sync()
	return screen
}

// ClearLine erases the contents of the line starting from (x,y) coordinate
// till the end of the line.
func (screen *Screen) ClearLine(x int, y int) *Screen {
	for i := x; i < screen.Width; i++ {
		termbox.SetCell(i, y, ' ', termbox.ColorDefault, termbox.ColorDefault)
	}
	termbox.Flush()

	return screen
}

func (screen *Screen) Flush() *Screen {
	termbox.Flush()
	return screen
}

// RenderLine takes the incoming string, tokenizes it to extract markup
// elements, and displays it all starting at (x,y) location.
func (screen *Screen) RenderLine(x int, y int, str string) {
	start, column := 0, 0

	for _, token := range screen.markup.Tokenize(str) {
		// First check if it's a tag. Tags are eaten up and not displayed.
		if screen.markup.IsTag(token) {
			continue
		}

		// Here comes the actual text: display it one character at a time.
		for i, char := range token {
			if !screen.markup.RightAligned {
				start = x + column
				column++
			} else {
				start = screen.Width - len(token) + i
			}
			termbox.SetCell(start, y, char, screen.markup.Foreground, screen.markup.Background)
		}
	}
}

func (screen *Screen) MoveCursorDown() {
	screen.Cursor.Line = screen.Cursor.Line + 1

}
func (screen *Screen) MoveCursorUp() {
	screen.Cursor.Line = screen.Cursor.Line - 1
}

func (screen *Screen) CursorPosition() int {
	return screen.Cursor.Line
}

func (screen *Screen) Render(column int, str string) {
	if !screen.cleared {
		screen.Clear()
	}
	for row, line := range strings.Split(str, "\n") {
		screen.RenderLine(column, row, line)
	}
}
