package ui

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/termbox"
	"github.com/moncho/dry/terminal"
)

type eventCallback func()

// A View is a region of the screen where text can be rendered. It maintains
// its own internal buffer and cursor position.
type View struct {
	name             string
	x0, y0, x1, y1   int      //view position in the screen
	width, height    int      //view width,height
	bufferX, bufferY int      //current position in the view buffer
	cursorX, cursorY int      //cursor position in the screen, valid values between 0 and x1, y1
	lines            [][]rune //the content buffer
	showCursor       bool

	newLineCallback eventCallback

	markup   *Markup
	theme    *ColorTheme
	renderer ScreenTextRenderer
}

// ViewSize returns the width and the height of the View.
func (v *View) ViewSize() (width, height int) {
	return v.width, v.height
}

// Name returns the name of the view.
func (v *View) Name() string {
	return v.name
}

// SetCursor sets the cursor position of the view at the given point,
// relative to the screen. An error is returned if the position is outside
// the screen limits.
func (v *View) setCursor(x, y int) error {
	maxX, maxY := v.ViewSize()
	if x < 0 || x >= maxX || y < 0 || y >= maxY {
		return invalidPointError(x, y)
	}
	v.cursorX = x
	v.cursorY = y
	return nil
}

// Cursor returns the cursor position of the view.
func (v *View) Cursor() (x, y int) {
	return v.cursorX, v.cursorY
}

// setPosition sets the origin position of the view's internal buffer,
// so the buffer starts to be printed from this point, which means that
// it is linked with the origin point of view. It can be used to
// implement Horizontal and Vertical scrolling with just incrementing
// or decrementing x and y.
func (v *View) setPosition(x, y int) error {
	if x < 0 || y < 0 {
		return errors.New("invalid point")
	}
	v.bufferX = x
	v.bufferY = y
	return nil
}

// Position returns the position in the view buffer.
func (v *View) Position() (x, y int) {
	return v.bufferX, v.bufferY
}

// Write appends a byte slice into the view's internal buffer, as defined
// by the io.Writer interface.
func (v *View) Write(p []byte) (n int, err error) {

	for _, ch := range bytes.Runes(p) {
		switch ch {
		case '\n':
			v.lines = append(v.lines, nil)
			v.newLineCallback()
		case '\r':
			nl := len(v.lines)
			if nl > 0 {
				v.lines[nl-1] = nil
			} else {
				v.lines = make([][]rune, 1)
			}
		default:
			nl := len(v.lines)
			if nl > 0 {
				v.lines[nl-1] = append(v.lines[nl-1], ch)
				//If the length of the line is higher than then view size
				//content goes to a new line
				if len(v.lines[nl-1]) >= v.width {
					v.lines = append(v.lines, nil)
				}
			} else {
				v.lines = append(v.lines, []rune{ch})
			}
		}
	}
	return len(p), nil
}

// realPosition returns the position in the internal buffer corresponding to the
// point (x, y) of the view.
func (v *View) realPosition(vx, vy int) (x, y int, err error) {
	vx = v.bufferX + vx
	vy = v.bufferY + vy

	if vx < 0 || vy < 0 {
		return 0, 0, invalidPointError(x, y)
	}

	if len(v.lines) == 0 {
		return vx, vy, nil
	}
	x = vx
	if vy < len(v.lines) {
		y = vy
	} else {
		y = vy - len(v.lines) + 1
	}

	return x, y, nil
}

// renderLine renders the given line, returns the number of screen lines used
func (v *View) renderLine(x int, y int, line string) (int, error) {
	lines := 1
	maxWidth, _ := v.ViewSize()
	if v.markup != nil {
		lines = renderLineWithMarkup(x, y, maxWidth, line, v.markup)
	} else {

		ansiClean := terminal.RemoveANSIEscapeCharacters(line)
		// Methods receives a single line, so just the first element
		// returned by the cleaner is considered
		if len(ansiClean) > 0 {
			lines = v.renderer.On(x, y).Render(string(ansiClean[0]))
		}
	}
	return lines, nil
}

// Clear empties the view's internal buffer.
func (v *View) Clear() {
	v.lines = nil
	v.clearRunes()
}

// clearRunes erases all the cells in the view.
func (v *View) clearRunes() {
	maxX, maxY := v.ViewSize()

	ActiveScreen.Fill(0, 0, maxX, maxY, ' ')
}

// Line returns a string with the line of the view's internal buffer
// at the position corresponding to the point (x, y).
func (v *View) Line(y int) (string, error) {
	_, y, err := v.realPosition(0, y)
	if err != nil {
		return "", err
	}

	if y < 0 || y >= len(v.lines) {
		return "", invalidPointError(0, y)
	}
	return string(v.lines[y]), nil
}

// Word returns a string with the word of the view's internal buffer
// at the position corresponding to the point (x, y).
func (v *View) Word(x, y int) (string, error) {
	x, y, err := v.realPosition(x, y)
	if err != nil {
		return "", err
	}

	if x < 0 || y < 0 || y >= len(v.lines) || x >= len(v.lines[y]) {
		return "", invalidPointError(x, y)
	}
	l := string(v.lines[y])
	nl := strings.LastIndexFunc(l[:x], indexFunc)
	if nl == -1 {
		nl = 0
	} else {
		nl++
	}
	nr := strings.IndexFunc(l[x:], indexFunc)
	if nr == -1 {
		nr = len(l)
	} else {
		nr += x
	}
	return l[nl:nr], nil
}

// CursorDown moves the cursor down one line
func (v *View) CursorDown() {
	cursorX, cursorY := v.Cursor()
	ox, bufferY := v.Position()
	if bufferY+cursorY <= len(v.lines) {
		if err := v.setCursor(cursorX, cursorY+1); err != nil {
			v.setPosition(ox, bufferY+1)
		}
	}
}

// CursorUp moves the cursor up one line
func (v *View) CursorUp() {
	ox, bufferY := v.Position()
	cursorX, cursorY := v.Cursor()
	if err := v.setCursor(cursorX, cursorY-1); err != nil && bufferY > 0 {
		v.setPosition(ox, bufferY-1)
	}
}

// PageDown moves the buffer position down by the length of the screen,
// at the end of buffer it also moves the cursor position to the bottom
// of the screen
func (v *View) PageDown() {
	_, cursorY := v.Cursor()
	bufferX, bufferY := v.Position()
	_, height := v.ViewSize()
	viewLength := len(v.lines)
	if bufferY+height+cursorY < viewLength {
		newOy := bufferY + height
		if newOy >= viewLength {
			v.setPosition(bufferX, viewLength)
		} else {
			v.setPosition(bufferX, newOy)
		}
		_, bufferY := v.Position()
		if bufferY >= viewLength-cursorY {
			v.CursorDown()
		}
	} else {
		v.CursorToBottom()
	}
}

// PageUp moves the buffer position up by the length of the screen,
// at the beginning of buffer it also moves the cursor position to the beginning
// of the screen
func (v *View) PageUp() {
	bufferX, bufferY := v.Position()
	cursorX, cursorY := v.Cursor()
	_, height := v.ViewSize()
	if err := v.setCursor(cursorX, cursorY-height); err != nil && bufferY > 0 {
		newOy := bufferY - height
		if newOy < 0 {
			v.setPosition(bufferX, 0)
		} else {
			v.setPosition(bufferX, newOy)
		}
	}
}

// CursorToBottom moves the cursor to the bottom of the view buffer
func (v *View) CursorToBottom() {
	v.bufferY = len(v.lines) - v.y1
	v.cursorY = v.y1
}

// CursorToTop moves the cursor to the top of the view buffer
func (v *View) CursorToTop() {
	v.bufferY = 0
	v.cursorY = 0
}

// MarkupSupport sets markup support in the view
func (v *View) MarkupSupport() {
	v.markup = NewMarkup(v.theme)
}

// NewMarkupView returns a new View with markup support
func NewMarkupView(name string, x0, y0, x1, y1 int, showCursor bool, theme *ColorTheme) *View {
	v := NewView(name, x0, y0, x1, y1, showCursor, theme)
	v.markup = NewMarkup(theme)

	return v
}

// NewView returns a new View
func NewView(name string, x0, y0, x1, y1 int, showCursor bool, theme *ColorTheme) *View {

	v := View{
		name:  name,
		x0:    x0,
		y0:    y0,
		x1:    x1,
		y1:    y1,
		width: x1 - x0,
		//last line is used by the cursor and for reading input, it is not used to
		//render view buffer
		height:          y1 - y0 - 1,
		showCursor:      showCursor,
		theme:           theme,
		newLineCallback: func() {},
	}

	v.renderer = NewRenderer(screenStyledRuneRenderer{ActiveScreen}).WithWidth(v.width).WithStyle(
		mkStyle(termbox.Attribute(theme.Fg), termbox.Attribute(theme.Bg)))

	return &v
}

// indexFunc allows to split lines by words taking into account spaces
// and 0.
func indexFunc(r rune) bool {
	return r == ' ' || r == 0
}

func invalidPointError(x, y int) error {
	_, file, line, _ := runtime.Caller(2)
	return fmt.Errorf(
		"Invalid point. x: %d, y: %d. Caller: %s, line: %d", x, y, file, line)
}
