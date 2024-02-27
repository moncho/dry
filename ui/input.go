package ui

import (
	"unicode/utf8"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/termbox"
)

const (
	preferredHorizontalThreshold = 5
	tabStopLength                = 8
	editBoxWidth                 = 30
)

// InputBox captures user input
type InputBox struct {
	text          []byte
	lineVOffset   int
	cursorBOffset int // cursor offset in bytes
	cursorVOffset int // visual cursor offset in termbox cells
	cursorCOffset int // cursor offset in unicode code points
	x, y          int // InputBox position in the screen
	output        chan<- string
	eventQueue    chan *tcell.EventKey
	screen        *Screen
}

// Draw draws the InputBox in the given location
func (eb *InputBox) Draw(x, y, w, h int) {
	eb.AdjustVOffset(w)

	eb.screen.Fill(x, y, w, h, ' ')
	t := eb.text
	lx := 0
	tabstop := 0
	for {
		rx := lx - eb.lineVOffset
		if len(t) == 0 {
			break
		}

		if lx == tabstop {
			tabstop += tabStopLength
		}

		if rx >= w {
			eb.screen.RenderRune(x+w-1, y, '→')
			break
		}

		r, size := utf8.DecodeRune(t)
		if r == '\t' {
			for ; lx < tabstop; lx++ {
				rx = lx - eb.lineVOffset
				if rx >= w {
					goto next
				}

				if rx >= 0 {
					eb.screen.RenderRune(x+rx, y, ' ')
				}
			}
		} else {
			if rx >= 0 {
				eb.screen.RenderRune(x+rx, y, r)
			}
			lx++
		}
	next:
		t = t[size:]
	}

	if eb.lineVOffset != 0 {
		eb.screen.RenderRune(x, y, '←')
	}
}

// AdjustVOffset adjusts line visual offset to a proper value depending on width
func (eb *InputBox) AdjustVOffset(width int) {
	ht := preferredHorizontalThreshold
	maxHThreshold := (width - 1) / 2
	if ht > maxHThreshold {
		ht = maxHThreshold
	}

	threshold := width - 1
	if eb.lineVOffset != 0 {
		threshold = width - ht
	}
	if eb.cursorVOffset-eb.lineVOffset >= threshold {
		eb.lineVOffset = eb.cursorVOffset + (ht - width + 1)
	}

	if eb.lineVOffset != 0 && eb.cursorVOffset-eb.lineVOffset < ht {
		eb.lineVOffset = eb.cursorVOffset - ht
		if eb.lineVOffset < 0 {
			eb.lineVOffset = 0
		}
	}
}

// MoveCursorTo moves the cursor
func (eb *InputBox) MoveCursorTo(boffset int) {
	eb.cursorBOffset = boffset
	eb.cursorVOffset, eb.cursorCOffset = vOffsetToCOffset(eb.text, boffset)
}

// RuneUnderCursor returns the rune from the inputbox where the cursor is
func (eb *InputBox) RuneUnderCursor() (rune, int) {
	return utf8.DecodeRune(eb.text[eb.cursorBOffset:])
}

// RuneBeforeCursor returns the rune from the inputbox placed before the cursor
func (eb *InputBox) RuneBeforeCursor() (rune, int) {
	return utf8.DecodeLastRune(eb.text[:eb.cursorBOffset])
}

// MoveCursorOneRuneBackward moves the cursor one rune backwards
func (eb *InputBox) MoveCursorOneRuneBackward() {
	if eb.cursorBOffset == 0 {
		return
	}
	_, size := eb.RuneBeforeCursor()
	eb.MoveCursorTo(eb.cursorBOffset - size)
}

// MoveCursorOneRuneForward moves the cursor one rune forward
func (eb *InputBox) MoveCursorOneRuneForward() {
	if eb.cursorBOffset == len(eb.text) {
		return
	}
	_, size := eb.RuneUnderCursor()
	eb.MoveCursorTo(eb.cursorBOffset + size)
}

// MoveCursorToBeginningOfTheLine moves the cursor to the beginning of the line
func (eb *InputBox) MoveCursorToBeginningOfTheLine() {
	eb.MoveCursorTo(0)
}

// MoveCursorToEndOfTheLine moves the cursor to the end of the line
func (eb *InputBox) MoveCursorToEndOfTheLine() {
	eb.MoveCursorTo(len(eb.text))
}

// Delete deletes the content of the inputbox
func (eb *InputBox) Delete() {
	if eb.cursorBOffset == 0 {
		return
	}
	eb.text = nil
}

// DeleteRuneBackward deletes a rune moving the cursor backwards
func (eb *InputBox) DeleteRuneBackward() {
	if eb.cursorBOffset == 0 {
		return
	}

	eb.MoveCursorOneRuneBackward()
	_, size := eb.RuneUnderCursor()
	eb.text = byteSliceRemove(eb.text, eb.cursorBOffset, eb.cursorBOffset+size)
}

// DeleteRuneForward deletes a rune and moving the cursor forward
func (eb *InputBox) DeleteRuneForward() {
	if eb.cursorBOffset == len(eb.text) {
		return
	}
	_, size := eb.RuneUnderCursor()
	eb.text = byteSliceRemove(eb.text, eb.cursorBOffset, eb.cursorBOffset+size)
}

// DeleteTheRestOfTheLine deletes the conent of the line where the cursor is
// from the cursor position until the end of the line
func (eb *InputBox) DeleteTheRestOfTheLine() {
	eb.text = eb.text[:eb.cursorBOffset]
}

// InsertRune adds the given rune to the inputbox
func (eb *InputBox) InsertRune(r rune) {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	eb.text = byteSliceInsert(eb.text, eb.cursorBOffset, buf[:n])
	eb.MoveCursorOneRuneForward()
}

// CursorX Please, keep in mind that cursor depends on the value of lineVOffset, which
// is being set on Draw() call, so.. call this method after Draw() one.
func (eb *InputBox) CursorX() int {
	return eb.cursorVOffset - eb.lineVOffset
}

// String returns the inputbox content as a string
func (eb *InputBox) String() string {
	return string(eb.text)
}

func (eb *InputBox) redrawAll() {

	eb.Draw(eb.x, eb.y, editBoxWidth, 1)
	eb.screen.ShowCursor(eb.x+eb.CursorX(), eb.y)

	eb.screen.Flush()
}

// Focus is set on the inputbox, it starts handling terminal events and responding
// to user actions.
func (eb *InputBox) Focus() {
	//TODO eb.screen.SetInputMode(termbox.InputEsc)

	eb.redrawAll()
mainloop:
	for ev := range eb.eventQueue {
		switch ev.Key() {
		case tcell.KeyEnter:
			break mainloop
		case tcell.KeyEsc:
			eb.Delete()
			break mainloop
		case tcell.KeyLeft, tcell.KeyCtrlB:
			eb.MoveCursorOneRuneBackward()
		case tcell.KeyRight, tcell.KeyCtrlF:
			eb.MoveCursorOneRuneForward()
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			eb.DeleteRuneBackward()
		case tcell.KeyDelete, tcell.KeyCtrlD:
			eb.DeleteRuneForward()
		case tcell.KeyTab:
			eb.InsertRune('\t')
		//TODO case tcell.KeySpace:
		//eb.InsertRune(' ')
		case tcell.KeyCtrlK:
			eb.DeleteTheRestOfTheLine()
		case tcell.KeyHome, tcell.KeyCtrlA:
			eb.MoveCursorToBeginningOfTheLine()
		case tcell.KeyEnd, tcell.KeyCtrlE:
			eb.MoveCursorToEndOfTheLine()
		default:
			if ev.Rune() != 0 {
				eb.InsertRune(ev.Rune())
			}
		}
		eb.redrawAll()
	}
	eb.output <- eb.String()
}

// NewInputBox creates an input box, located at position x,y in the screen.
func NewInputBox(x, y int, prompt string, output chan<- string, keyboardQueue chan *tcell.EventKey, theme *ColorTheme, screen *Screen) *InputBox {
	width := screen.Dimensions().Width
	//TODO use color from the theme for the prompt
	r := NewRenderer(screenStyledRuneRenderer{screen}).On(x, y).WithWidth(width).WithStyle(
		mkStyle(termbox.ColorYellow, termbox.Attribute(theme.Bg)))
	r.Render(prompt)
	screen.Flush()
	return &InputBox{x: x + len(prompt), y: y, output: output, eventQueue: keyboardQueue, screen: screen}
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
