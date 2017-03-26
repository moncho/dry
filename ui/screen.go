package ui

import (
	"strings"
	"sync"
	"time"

	termui "github.com/gizak/termui"
	"github.com/nsf/termbox-go"
)

//ActiveScreen is the currently active screen
var ActiveScreen *Screen

// Screen is thin wrapper aroung Termbox library to provide basic display
// capabilities as required by dry.
type Screen struct {
	markup   *Markup // Pointer to markup processor (gets created by screen).
	pausedAt *time.Time
	Cursor   *Cursor // Pointer to cursor (gets created by screen).
	sync.RWMutex
	theme      *ColorTheme
	Dimensions *ScreenDimension
}

//Cursor represents the cursor position on the screen
type Cursor struct {
	line int
	sync.RWMutex
}

//NewScreen initializes Termbox, creates screen along with layout and markup, and
//calculates current screen dimensions. Once created, the screen is
//ready for display.
func NewScreen(theme *ColorTheme) (*Screen, error) {

	sd := screenDimensions()

	termbox.SetOutputMode(termbox.Output256)
	screen := &Screen{}
	screen.markup = NewMarkup(theme)
	screen.Cursor = &Cursor{line: 0}
	screen.theme = theme
	screen.Dimensions = sd
	ActiveScreen = screen
	return screen, nil
}

func screenDimensions() *ScreenDimension {
	w, h := termbox.Size()
	return &ScreenDimension{Width: w, Height: h}
}

// Close gets called upon program termination to close the Termbox.
func (screen *Screen) Close() *Screen {
	termbox.Close()
	return screen
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

// ClearLine erases the contents of the line starting from (x,y) coordinate
// till the end of the line.
func (screen *Screen) ClearLine(x int, y int) *Screen {
	screen.RLock()
	defer screen.RUnlock()
	for i := x; i < ActiveScreen.Dimensions.Width; i++ {
		termbox.SetCell(i, y, ' ', termbox.Attribute(screen.theme.Fg), termbox.Attribute(screen.theme.Bg))
	}
	screen.Flush()

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

//Position tells on which screen line the cursor is
func (cursor *Cursor) Position() int {
	cursor.RLock()
	defer cursor.RUnlock()
	return cursor.line
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

// RenderLine takes the incoming string, tokenizes it to extract markup
// elements, and displays it all starting at (x,y) location.
func (screen *Screen) RenderLine(x int, y int, str string) {
	screen.Lock()
	defer screen.Unlock()

	start, column := 0, 0
	for _, token := range Tokenize(str, SupportedTags) {
		// First check if it's a tag. Tags are eaten up and not displayed.
		if screen.markup.IsTag(token) {
			continue
		}

		// Here comes the actual text: displays it one character at a time.
		for i, char := range token {
			if !screen.markup.RightAligned {
				start = x + column
				column++
			} else {
				start = ActiveScreen.Dimensions.Width - len(token) + i
			}
			termbox.SetCell(start, y, char, screen.markup.Foreground, screen.markup.Background)
		}
	}
}

//RenderLineWithBackGround does what RenderLine does but rendering the line
//with the given background color
func (screen *Screen) RenderLineWithBackGround(x int, y int, str string, bgColor Color) {
	screen.Lock()
	defer screen.Unlock()
	start, column := 0, 0
	if x > 0 {
		fill(0, y, x, y, termbox.Cell{Ch: ' ', Bg: termbox.Attribute(bgColor)})
	}
	for _, token := range Tokenize(str, SupportedTags) {
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
				start = ActiveScreen.Dimensions.Width - len(token) + i
			}
			termbox.SetCell(start, y, char, screen.markup.Foreground, termbox.Attribute(bgColor))
		}
	}
	fill(start+1, y, ActiveScreen.Dimensions.Width, y, termbox.Cell{Ch: ' ', Bg: termbox.Attribute(bgColor)})
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

//Reset sets the cursor in the initial position
func (cursor *Cursor) Reset() {
	cursor.Lock()
	defer cursor.Unlock()
	cursor.line = 0
}

//ScrollCursorDown moves the cursor to the line below the current one
func (cursor *Cursor) ScrollCursorDown() {
	cursor.Lock()
	defer cursor.Unlock()
	cursor.line = cursor.line + 1
}

//ScrollCursorUp moves the cursor to the line above the current one
func (cursor *Cursor) ScrollCursorUp() {
	cursor.Lock()
	defer cursor.Unlock()
	if cursor.line > 0 {
		cursor.line = cursor.line - 1
	} else {
		cursor.line = 0
	}
}

//ScrollTo moves the cursor to the given line
func (cursor *Cursor) ScrollTo(pos int) {
	cursor.Lock()
	defer cursor.Unlock()
	cursor.line = pos

}

func toTmAttr(x termui.Attribute) termbox.Attribute {
	return termbox.Attribute(x)
}
