package ui

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/moncho/dry/terminal"
	"github.com/nsf/termbox-go"
)

// A View is a region of the screen where text can be rendered. It maintains
//its own internal buffer and cursor position.
type View struct {
	name             string
	x0, y0, x1, y1   int      //view position in the screen
	width, height    int      //view width,height
	bufferX, bufferY int      //current position in the view buffer
	cursorX, cursorY int      //cursor position in the screen, valid values between 0 and x1, y1
	lines            [][]rune //the content buffer
	showCursor       bool

	tainted bool // marks if the viewBuffer must be updated

	markup *Markup
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
	v.tainted = true

	for _, ch := range bytes.Runes(p) {
		switch ch {
		case '\n':
			v.lines = append(v.lines, nil)
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

// Render renders the view buffer contents.
func (v *View) Render() error {
	_, maxY := v.ViewSize()
	x, y := 0, 0
	for _, vline := range v.lines[v.bufferY:] {
		if y > maxY {
			break
		}
		v.renderLine(x, y, string(vline))
		y++
	}
	if v.showCursor {
		v.drawCursor()
	}
	return nil
}

//calculateCursorPosition gives the cursor position
//from the beginning of the view
func (v *View) calculateCursorPosition() (int, int) {
	return v.x0 + v.cursorX, v.y0 + v.cursorY
}

func (v *View) drawCursor() {
	cursorX, cursorY := v.calculateCursorPosition()

	_, ry, _ := v.realPosition(cursorX, cursorY)

	if ry <= len(v.lines) {
		termbox.SetCursor(cursorX, cursorY)
	}
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

//renderLine renders the given line, returns the number of screen lines used
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
			_, lines = renderString(x, y, maxWidth, string(ansiClean[0]), termbox.ColorDefault, termbox.ColorDefault)
		}
	}
	return lines, nil
}

// Clear empties the view's internal buffer.
func (v *View) Clear() {
	v.tainted = true
	v.lines = nil
	v.clearRunes()
}

// clearRunes erases all the cells in the view.
func (v *View) clearRunes() {
	maxX, maxY := v.ViewSize()
	for x := 0; x < maxX; x++ {
		for y := 0; y < maxY; y++ {
			termbox.SetCell(v.x0+x+1, v.y0+y+1, ' ',
				termbox.ColorDefault, termbox.ColorDefault)
		}
	}
}

// writeRune writes a rune into the view's internal buffer, at the
// position corresponding to the point (x, y). The length of the internal
// buffer is increased if the point is out of bounds. Overwrite mode is
// governed by the value of View.overwrite.
func (v *View) writeRune(x, y int, ch rune) error {
	v.tainted = true

	x, y, err := v.realPosition(x, y)
	if err != nil {
		return err
	}

	if x < 0 || y < 0 {
		return invalidPointError(x, y)
	}

	if y >= len(v.lines) {
		s := make([][]rune, y-len(v.lines)+1)
		v.lines = append(v.lines, s...)
	}

	olen := len(v.lines[y])
	if x >= len(v.lines[y]) {
		s := make([]rune, x-len(v.lines[y])+1)
		v.lines[y] = append(v.lines[y], s...)
	}

	if x < olen {
		v.lines[y] = append(v.lines[y], '\x00')
		copy(v.lines[y][x+1:], v.lines[y][x:])
	}
	v.lines[y][x] = ch
	return nil
}

// deleteRune removes a rune from the view's internal buffer, at the
// position corresponding to the point (x, y).
func (v *View) deleteRune(x, y int) error {
	v.tainted = true

	x, y, err := v.realPosition(x, y)
	if err != nil {
		return err
	}

	if x < 0 || y < 0 || y >= len(v.lines) || x >= len(v.lines[y]) {
		return invalidPointError(x, y)
	}
	v.lines[y] = append(v.lines[y][:x], v.lines[y][x+1:]...)
	return nil
}

// mergeLines merges the lines "y" and "y+1" if possible.
func (v *View) mergeLines(y int) error {
	v.tainted = true

	_, y, err := v.realPosition(0, y)
	if err != nil {
		return err
	}

	if y < 0 || y >= len(v.lines) {
		return invalidPointError(0, y)
	}

	if y < len(v.lines)-1 { // otherwise we don't need to merge anything
		v.lines[y] = append(v.lines[y], v.lines[y+1]...)
		v.lines = append(v.lines[:y+1], v.lines[y+2:]...)
	}
	return nil
}

// breakLine breaks a line of the internal buffer at the position corresponding
// to the point (x, y).
func (v *View) breakLine(x, y int) error {
	v.tainted = true

	x, y, err := v.realPosition(x, y)
	if err != nil {
		return err
	}

	if y < 0 || y >= len(v.lines) {
		return invalidPointError(x, y)
	}

	var left, right []rune
	if x < len(v.lines[y]) { // break line
		left = make([]rune, len(v.lines[y][:x]))
		copy(left, v.lines[y][:x])
		right = make([]rune, len(v.lines[y][x:]))
		copy(right, v.lines[y][x:])
	} else { // new empty line
		left = v.lines[y]
	}

	lines := make([][]rune, len(v.lines)+1)
	lines[y] = left
	lines[y+1] = right
	copy(lines, v.lines[:y])
	copy(lines[y+2:], v.lines[y+1:])
	v.lines = lines
	return nil
}

// ViewBuffer returns a string with the contents of the view's buffer that is
// showed to the user
func (v *View) ViewBuffer() string {
	result := make([]string, 0, len(v.lines))
	for _, l := range v.lines {
		line := string(l)
		strings.Replace(line, "\x00", " ", -1)
		result = append(result, line)
	}
	return strings.Join(result, "\n")
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
		nl = nl + 1
	}
	nr := strings.IndexFunc(l[x:], indexFunc)
	if nr == -1 {
		nr = len(l)
	} else {
		nr = nr + x
	}
	return string(l[nl:nr]), nil
}

//CursorDown moves the cursor down one line
func (v *View) CursorDown() {
	cursorX, cursorY := v.Cursor()
	ox, bufferY := v.Position()
	if bufferY+cursorY <= len(v.lines) {
		if err := v.setCursor(cursorX, cursorY+1); err != nil {
			v.setPosition(ox, bufferY+1)
		}
	}
}

//CursorUp moves the cursor up one line
func (v *View) CursorUp() {
	ox, bufferY := v.Position()
	cursorX, cursorY := v.Cursor()
	if err := v.setCursor(cursorX, cursorY-1); err != nil && bufferY > 0 {
		v.setPosition(ox, bufferY-1)
	}
}

//PageDown moves the buffer position down by the length of the screen,
//at the end of buffer it also moves the cursor position to the bottom
//of the screen
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

//PageUp moves the buffer position up by the length of the screen,
//at the beginning of buffer it also moves the cursor position to the beginning
//of the screen
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

//CursorToBottom moves the cursor to the bottom of the view buffer
func (v *View) CursorToBottom() {
	v.bufferY = len(v.lines) - v.y1
	v.cursorY = v.y1
}

//CursorToTop moves the cursor to the top of the view buffer
func (v *View) CursorToTop() {
	v.bufferY = 0
	v.cursorY = 0
}

//MarkupSupport sets markup support in the view
func (v *View) MarkupSupport() {
	v.markup = NewMarkup()
}

// indexFunc allows to split lines by words taking into account spaces
// and 0.
func indexFunc(r rune) bool {
	return r == ' ' || r == 0
}

// NewView returns a new View
func NewView(name string, x0, y0, x1, y1 int, showCursor bool) *View {
	v := &View{
		name:  name,
		x0:    x0,
		y0:    y0,
		x1:    x1,
		y1:    y1,
		width: x1 - x0,
		//last line is used by the cursor and for reading input, it is not used to
		//render view buffer
		height:     y1 - y0 - 1,
		tainted:    true,
		showCursor: showCursor,
	}

	return v
}

// NewMarkupView returns a new View with markup support
func NewMarkupView(name string, x0, y0, x1, y1 int, showCursor bool) *View {
	v := NewView(name, x0, y0, x1, y1, showCursor)
	v.markup = NewMarkup()

	return v
}

func invalidPointError(x, y int) error {
	_, file, line, _ := runtime.Caller(2)
	return fmt.Errorf(
		"Invalid point. x: %d, y: %d. Caller: %s, line: %d", x, y, file, line)
}
