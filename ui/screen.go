package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/termbox"

	"github.com/gizak/termui"
)

// ActiveScreen is the currently active screen
var ActiveScreen *Screen

// Screen is where text is rendered
type Screen struct {
	markup     *Markup
	cursor     *Cursor
	theme      *ColorTheme
	screen     tcell.Screen
	themeStyle tcell.Style

	sync.RWMutex
	closing    bool
	dimensions *Dimensions
}

// NewScreen creates a new Screen and sets the ActiveScreen
func NewScreen(theme *ColorTheme) (*Screen, error) {
	s, err := initScreen()
	if err != nil {
		return nil, fmt.Errorf("init screen: %w", err)
	}
	screen := &Screen{}
	screen.markup = NewMarkup(theme)
	screen.cursor = &Cursor{pos: 0, downwards: true}
	screen.theme = theme
	screen.dimensions = screenDimensions(s)
	screen.screen = s
	screen.themeStyle = mkStyle(
		screen.markup.Foreground,
		screen.markup.Background)
	ActiveScreen = screen

	return screen, nil
}

// Close gets called upon program termination to close
func (screen *Screen) Close() *Screen {
	screen.Lock()
	screen.closing = true
	screen.Unlock()
	screen.screen.Fini()
	return screen
}

// Closing returns true if this this screen is closing
func (screen *Screen) Closing() bool {
	screen.RLock()
	defer screen.RUnlock()
	return screen.closing
}

// Cursor returns the screen cursor
func (screen *Screen) Cursor() *Cursor {
	return screen.cursor
}

// Dimensions returns the screen dimensions
func (screen *Screen) Dimensions() *Dimensions {
	return screen.dimensions
}

// Fill fills the squared portion of the screen delimited by the given
// positions with the provided rune. It uses this screen style.
func (screen *Screen) Fill(x, y, w, h int, r rune) {
	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			screen.RenderRune(x+lx, y+ly, r)
		}
	}
}

// RenderRune renders the given rune on the given pos
func (screen *Screen) RenderRune(x, y int, r rune) {
	screen.screen.SetCell(x, y, screen.themeStyle, r)

}

// Resize resizes this screen
func (screen *Screen) Resize() {
	w, h := screen.screen.Size()

	if w > 0 && h > 0 {
		screen.Lock()
		screen.dimensions.Width, screen.dimensions.Height = w, h
		screen.Unlock()
	}
}

// Clear makes the entire screen blank using default background color.
func (screen *Screen) Clear() *Screen {
	screen.Lock()
	defer screen.Unlock()
	st := mkStyle(
		termbox.Attribute(screen.theme.Fg), termbox.Attribute(screen.theme.Bg))
	screen.screen.Fill(' ', st)
	return screen
}

// ClearAndFlush cleares the screen and then flushes internal buffers
func (screen *Screen) ClearAndFlush() *Screen {
	screen.Clear()
	screen.Flush()
	return screen
}

// Sync is like flsuh but it ensures that whatever internal states are
// synchronized before flushing content to the screen.
func (screen *Screen) Sync() *Screen {
	screen.Lock()
	defer screen.Unlock()
	screen.screen.Sync()
	return screen
}

// ColorTheme changes the color theme of the screen to the given one.
func (screen *Screen) ColorTheme(theme *ColorTheme) *Screen {
	screen.Lock()
	defer screen.Unlock()
	screen.markup = NewMarkup(theme)
	return screen
}

// HideCursor hides the cursor.
func (screen *Screen) HideCursor() {
	screen.screen.HideCursor()
}

// Flush makes all the content visible on the display.
func (screen *Screen) Flush() *Screen {
	screen.Lock()
	defer screen.Unlock()
	screen.screen.Show()
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
				s := mkStyle(toTmAttr(c.Fg), toTmAttr(c.Bg))
				screen.screen.SetCell(p.X, p.Y, s, c.Ch)
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
		//Tags are not rendered
		if screen.markup.IsTag(token) {
			continue
		}

		//Render one character at a time
		for _, char := range token {
			s := mkStyle(
				screen.markup.Foreground,
				screen.markup.Background)
			screen.screen.SetCell(column, y, s, char)
			column++
		}
	}
}

// Render renders the given content starting from the given row
func (screen *Screen) Render(row int, str string) {
	screen.RenderAtColumn(0, row, str)
}

// RenderAtColumn renders the given content starting from
// the given row at the given column
func (screen *Screen) RenderAtColumn(column, initialRow int, str string) {
	for row, line := range strings.Split(str, "\n") {
		screen.RenderLine(column, initialRow+row, line)
	}
}

// ShowCursor shows the cursor on the given position
func (screen *Screen) ShowCursor(x, y int) {
	screen.screen.ShowCursor(x, y)
}

func toTmAttr(x termui.Attribute) termbox.Attribute {
	return termbox.Attribute(x)
}
