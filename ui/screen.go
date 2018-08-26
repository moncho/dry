package ui

import (
	"strings"
	"sync"

	termui "github.com/gizak/termui"
	"github.com/nsf/termbox-go"
)

//ActiveScreen is the currently active screen
var ActiveScreen *Screen

// Screen is thin wrapper aroung Termbox library to provide basic display
// capabilities as required by dry.
type Screen struct {
	markup *Markup // Pointer to markup processor (gets created by screen).
	Cursor *Cursor // Pointer to cursor (gets created by screen).
	sync.RWMutex
	theme      *ColorTheme
	Dimensions *Dimensions
	closing    bool
}

//NewScreen initializes Termbox, creates screen along with layout and markup, and
//calculates current screen dimensions. Once created, the screen is
//ready for display.
func NewScreen(theme *ColorTheme) (*Screen, error) {

	sd := screenDimensions()

	termbox.SetOutputMode(termbox.Output256)
	screen := &Screen{}
	screen.markup = NewMarkup(theme)
	screen.Cursor = &Cursor{pos: 0, downwards: true}
	screen.theme = theme
	screen.Dimensions = sd
	ActiveScreen = screen
	return screen, nil
}

// Close gets called upon program termination to close the Termbox.
func (screen *Screen) Close() *Screen {
	screen.closing = true
	screen.Lock()
	defer screen.Unlock()
	termbox.Close()
	return screen
}

// Closing returns true if this this screen is closing
func (screen *Screen) Closing() bool {
	return screen.closing
}

// Resize recalculates active screen dimensions.
func Resize() {
	termbox.Sync()
	w, h := termbox.Size()
	if w > 0 && h > 0 {
		ActiveScreen.Dimensions.Width, ActiveScreen.Dimensions.Height = termbox.Size()
	}
}

//Clear makes the entire screen blank using default background color.
func (screen *Screen) Clear() *Screen {
	screen.Lock()
	defer screen.Unlock()
	termbox.Clear(termbox.Attribute(screen.theme.Fg), termbox.Attribute(screen.theme.Bg))
	return screen
}

//ClearAndFlush cleares the screen and then flushes internal buffers
func (screen *Screen) ClearAndFlush() *Screen {
	screen.Clear()
	screen.Flush()
	return screen
}

// Sync forces a complete resync between the termbox and a terminal.
func (screen *Screen) Sync() *Screen {
	screen.Lock()
	defer screen.Unlock()
	termbox.Sync()
	return screen
}

//ColorTheme changes the color theme of the screen to the given one.
func (screen *Screen) ColorTheme(theme *ColorTheme) *Screen {
	screen.Lock()
	defer screen.Unlock()
	screen.markup = NewMarkup(theme)
	return screen
}

//Flush synchronizes the internal buffer with the terminal.
func (screen *Screen) Flush() *Screen {
	screen.Lock()
	defer screen.Unlock()
	termbox.Flush()
	return screen
}

// RenderBufferer renders all Bufferer in the given order from left to right,
// right could overlap on left ones.
// This allows usage of termui widgets.
func (screen *Screen) RenderBufferer(bs ...termui.Bufferer) {
	screen.Lock()
	defer screen.Unlock()
	for _, b := range bs {
		buf := b.Buffer()
		// set cels in buf
		for p, c := range buf.CellMap {
			if p.In(buf.Area) {
				termbox.SetCell(p.X, p.Y, c.Ch, toTmAttr(c.Fg), toTmAttr(c.Bg))
			}
		}
	}
}

// RenderLine renders the given string, removing markup elements, at the given location.
func (screen *Screen) RenderLine(x int, y int, str string) {
	screen.Lock()
	defer screen.Unlock()

	column := x
	for _, token := range Tokenize(str, SupportedTags) {
		// First check if it's a tag. Tags are eaten up and not displayed.
		if screen.markup.IsTag(token) {
			continue
		}

		// Here comes the actual text: displays it one character at a time.
		for _, char := range token {
			termbox.SetCell(column, y, char, screen.markup.Foreground, screen.markup.Background)
			column++
		}
	}
}

//Render renders the given content starting from the given row
func (screen *Screen) Render(row int, str string) {
	screen.RenderAtColumn(0, row, str)
}

//RenderAtColumn renders the given content starting from
//the given row at the given column
func (screen *Screen) RenderAtColumn(column, initialRow int, str string) {
	for row, line := range strings.Split(str, "\n") {
		screen.RenderLine(column, initialRow+row, line)
	}
}

//RenderRenderer renders the given renderer starting from the given row
func (screen *Screen) RenderRenderer(row int, renderer Renderer) {
	screen.Render(row, renderer.Render())
}

func toTmAttr(x termui.Attribute) termbox.Attribute {
	return termbox.Attribute(x)
}

func screenDimensions() *Dimensions {
	w, h := termbox.Size()
	return &Dimensions{Width: w, Height: h}
}
