package ui

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/termbox"
	"github.com/gizak/termui"
)

// OptimizedScreen is an improved version of Screen with reduced lock contention
type OptimizedScreen struct {
	markup     *Markup
	cursor     *Cursor
	theme      *ColorTheme
	screen     tcell.Screen
	themeStyle tcell.Style

	// Separate locks for different concerns
	renderLock sync.Mutex   // For rendering operations only
	configLock sync.RWMutex // For theme and markup changes

	// Atomic fields for lock-free access
	closing    int64        // atomic bool (0=false, 1=true)
	dimensions atomic.Value // *Dimensions
}

// NewOptimizedScreen creates a new OptimizedScreen
func NewOptimizedScreen(theme *ColorTheme) (*OptimizedScreen, error) {
	s, err := initScreen()
	if err != nil {
		return nil, fmt.Errorf("init screen: %w", err)
	}

	screen := &OptimizedScreen{}
	screen.markup = NewMarkup(theme)
	screen.cursor = &Cursor{pos: 0, downwards: true}
	screen.theme = theme
	screen.dimensions.Store(screenDimensions(s))
	screen.screen = s
	screen.themeStyle = mkStyle(
		screen.markup.Foreground,
		screen.markup.Background)

	return screen, nil
}

// Close gets called upon program termination to close
func (screen *OptimizedScreen) Close() ScreenRenderer {
	atomic.StoreInt64(&screen.closing, 1)
	screen.screen.Fini()
	return screen
}

// Closing returns true if this screen is closing (lock-free)
func (screen *OptimizedScreen) Closing() bool {
	return atomic.LoadInt64(&screen.closing) == 1
}

// Cursor returns the screen cursor
func (screen *OptimizedScreen) Cursor() *Cursor {
	return screen.cursor
}

// Dimensions returns the screen dimensions (lock-free)
func (screen *OptimizedScreen) Dimensions() *Dimensions {
	return screen.dimensions.Load().(*Dimensions)
}

// Resize resizes this screen
func (screen *OptimizedScreen) Resize() {
	w, h := screen.screen.Size()
	if w > 0 && h > 0 {
		newDims := &Dimensions{Width: w, Height: h}
		screen.dimensions.Store(newDims)
	}
}

// Clear makes the entire screen blank using default background color.
func (screen *OptimizedScreen) Clear() ScreenRenderer {
	screen.renderLock.Lock()
	defer screen.renderLock.Unlock()

	// Read theme without holding render lock
	screen.configLock.RLock()
	fg, bg := screen.theme.Fg, screen.theme.Bg
	screen.configLock.RUnlock()

	st := mkStyle(termbox.Attribute(fg), termbox.Attribute(bg))
	screen.screen.Fill(' ', st)
	return screen
}

// Sync synchronizes internal states before flushing content
func (screen *OptimizedScreen) Sync() ScreenRenderer {
	screen.renderLock.Lock()
	defer screen.renderLock.Unlock()
	screen.screen.Sync()
	return screen
}

// Flush makes all the content visible on the display
func (screen *OptimizedScreen) Flush() ScreenRenderer {
	screen.renderLock.Lock()
	defer screen.renderLock.Unlock()
	screen.screen.Show()
	return screen
}

// ColorTheme changes the color theme of the screen
func (screen *OptimizedScreen) ColorTheme(theme *ColorTheme) ScreenRenderer {
	screen.configLock.Lock()
	defer screen.configLock.Unlock()
	screen.markup = NewMarkup(theme)
	screen.theme = theme
	screen.themeStyle = mkStyle(
		screen.markup.Foreground,
		screen.markup.Background)
	return screen
}

// RenderBufferer renders all Bufferer in the given order
func (screen *OptimizedScreen) RenderBufferer(bs ...termui.Bufferer) {
	if len(bs) == 0 {
		return
	}

	// Pre-process all buffers outside the render lock
	type cellOp struct {
		x, y  int
		char  rune
		style tcell.Style
	}

	var ops []cellOp
	for _, b := range bs {
		buf := b.Buffer()
		for p, c := range buf.CellMap {
			if p.In(buf.Area) {
				ops = append(ops, cellOp{
					x: p.X, y: p.Y,
					char:  c.Ch,
					style: mkStyle(toTmAttr(c.Fg), toTmAttr(c.Bg)),
				})
			}
		}
	}

	// Apply all operations under a single lock
	screen.renderLock.Lock()
	defer screen.renderLock.Unlock()

	for _, op := range ops {
		screen.screen.SetCell(op.x, op.y, op.style, op.char)
	}
}

// RenderLine renders the given string at the given location
func (screen *OptimizedScreen) RenderLine(x int, y int, str string) {
	// Tokenize and prepare render operations outside lock
	tokens := Tokenize(str, SupportedTags)

	screen.configLock.RLock()
	fg, bg := screen.markup.Foreground, screen.markup.Background
	isTag := screen.markup.IsTag
	screen.configLock.RUnlock()

	style := mkStyle(fg, bg)

	type renderOp struct {
		x    int
		char rune
	}

	var ops []renderOp
	column := x

	for _, token := range tokens {
		if isTag(token) {
			continue
		}

		for _, char := range token {
			ops = append(ops, renderOp{x: column, char: char})
			column++
		}
	}

	// Apply all operations under a single lock
	screen.renderLock.Lock()
	defer screen.renderLock.Unlock()

	for _, op := range ops {
		screen.screen.SetCell(op.x, y, style, op.char)
	}
}

// Fill fills the squared portion of the screen with the provided rune
func (screen *OptimizedScreen) Fill(x, y, w, h int, r rune) {
	screen.configLock.RLock()
	style := screen.themeStyle
	screen.configLock.RUnlock()

	screen.renderLock.Lock()
	defer screen.renderLock.Unlock()

	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			screen.screen.SetCell(x+lx, y+ly, style, r)
		}
	}
}

// RenderRune renders the given rune on the given position
func (screen *OptimizedScreen) RenderRune(x, y int, r rune) {
	screen.configLock.RLock()
	style := screen.themeStyle
	screen.configLock.RUnlock()

	screen.renderLock.Lock()
	screen.screen.SetCell(x, y, style, r)
	screen.renderLock.Unlock()
}

// ClearAndFlush clears the screen and then flushes internal buffers
func (screen *OptimizedScreen) ClearAndFlush() ScreenRenderer {
	screen.Clear()
	screen.Flush()
	return screen
}

// Render renders the given content starting from the given row
func (screen *OptimizedScreen) Render(row int, str string) {
	screen.RenderAtColumn(0, row, str)
}

// RenderAtColumn renders the given content starting from the given row at the given column
func (screen *OptimizedScreen) RenderAtColumn(column, initialRow int, str string) {
	lines := strings.Split(str, "\n")

	// Batch render all lines in a single lock
	screen.configLock.RLock()
	fg, bg := screen.markup.Foreground, screen.markup.Background
	isTag := screen.markup.IsTag
	screen.configLock.RUnlock()

	style := mkStyle(fg, bg)

	type lineOp struct {
		x, y int
		char rune
	}

	var ops []lineOp

	for rowOffset, line := range lines {
		currentRow := initialRow + rowOffset
		currentCol := column

		tokens := Tokenize(line, SupportedTags)
		for _, token := range tokens {
			if isTag(token) {
				continue
			}

			for _, char := range token {
				ops = append(ops, lineOp{x: currentCol, y: currentRow, char: char})
				currentCol++
			}
		}
	}

	screen.renderLock.Lock()
	defer screen.renderLock.Unlock()

	for _, op := range ops {
		screen.screen.SetCell(op.x, op.y, style, op.char)
	}
}

// HideCursor hides the cursor
func (screen *OptimizedScreen) HideCursor() {
	screen.screen.HideCursor()
}

// ShowCursor shows the cursor on the given position
func (screen *OptimizedScreen) ShowCursor(x, y int) {
	screen.screen.ShowCursor(x, y)
}

// func toTmAttr(x termui.Attribute) termbox.Attribute {
// 	return termbox.Attribute(x)
// }
