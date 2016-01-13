package ui

import (
	"unicode/utf8"

	"github.com/nsf/termbox-go"
)

const (
	preferredHorizontalThreshold = 5
	tabstopLength                = 8
	editBoxWidth                 = 30
	coldef                       = termbox.ColorDefault
)

//InputBox captures user input
type InputBox struct {
	text          []byte
	lineVOffset   int
	cursorBOffset int // cursor offset in bytes
	cursorVOffset int // visual cursor offset in termbox cells
	cursorCOffset int // cursor offset in unicode code points
	x, y          int // InputBox position in the screen
	output        chan<- string
	eventQueue    chan termbox.Event
}

//Draw draws the InputBox in the given location
func (eb *InputBox) Draw(x, y, w, h int) {
	eb.AdjustVOffset(w)

	fill(x, y, w, h, termbox.Cell{Ch: ' '})

	t := eb.text
	lx := 0
	tabstop := 0
	for {
		rx := lx - eb.lineVOffset
		if len(t) == 0 {
			break
		}

		if lx == tabstop {
			tabstop += tabstopLength
		}

		if rx >= w {
			termbox.SetCell(x+w-1, y, '→',
				coldef, coldef)
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
					termbox.SetCell(x+rx, y, ' ', coldef, coldef)
				}
			}
		} else {
			if rx >= 0 {
				termbox.SetCell(x+rx, y, r, coldef, coldef)
			}
			lx++
		}
	next:
		t = t[size:]
	}

	if eb.lineVOffset != 0 {
		termbox.SetCell(x, y, '←', coldef, coldef)
	}
}

//AdjustVOffset adjusts line visual offset to a proper value depending on width
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

//MoveCursorTo moves the cursor
func (eb *InputBox) MoveCursorTo(boffset int) {
	eb.cursorBOffset = boffset
	eb.cursorVOffset, eb.cursorCOffset = vOffsetToCOffset(eb.text, boffset)
}

//RuneUnderCursor returns the rune from the inputbox where the cursor is
func (eb *InputBox) RuneUnderCursor() (rune, int) {
	return utf8.DecodeRune(eb.text[eb.cursorBOffset:])
}

//RuneBeforeCursor returns the rune from the inputbox placed before the cursor
func (eb *InputBox) RuneBeforeCursor() (rune, int) {
	return utf8.DecodeLastRune(eb.text[:eb.cursorBOffset])
}

//MoveCursorOneRuneBackward moves the cursor one rune backwards
func (eb *InputBox) MoveCursorOneRuneBackward() {
	if eb.cursorBOffset == 0 {
		return
	}
	_, size := eb.RuneBeforeCursor()
	eb.MoveCursorTo(eb.cursorBOffset - size)
}

//MoveCursorOneRuneForward moves the cursor one rune forward
func (eb *InputBox) MoveCursorOneRuneForward() {
	if eb.cursorBOffset == len(eb.text) {
		return
	}
	_, size := eb.RuneUnderCursor()
	eb.MoveCursorTo(eb.cursorBOffset + size)
}

//MoveCursorToBeginningOfTheLine moves the cursor to the beginning of the line
func (eb *InputBox) MoveCursorToBeginningOfTheLine() {
	eb.MoveCursorTo(0)
}

//MoveCursorToEndOfTheLine moves the cursor to the end of the line
func (eb *InputBox) MoveCursorToEndOfTheLine() {
	eb.MoveCursorTo(len(eb.text))
}

//Delete deletes the content of the inputbox
func (eb *InputBox) Delete() {
	if eb.cursorBOffset == 0 {
		return
	}
	eb.text = nil
}

//DeleteRuneBackward deletes a rune moving the cursor backwards
func (eb *InputBox) DeleteRuneBackward() {
	if eb.cursorBOffset == 0 {
		return
	}

	eb.MoveCursorOneRuneBackward()
	_, size := eb.RuneUnderCursor()
	eb.text = byteSliceRemove(eb.text, eb.cursorBOffset, eb.cursorBOffset+size)
}

//DeleteRuneForward deletes a rune and moving the cursor forward
func (eb *InputBox) DeleteRuneForward() {
	if eb.cursorBOffset == len(eb.text) {
		return
	}
	_, size := eb.RuneUnderCursor()
	eb.text = byteSliceRemove(eb.text, eb.cursorBOffset, eb.cursorBOffset+size)
}

//DeleteTheRestOfTheLine deletes the conent of the line where the cursor is
//from the cursor position until the end of the line
func (eb *InputBox) DeleteTheRestOfTheLine() {
	eb.text = eb.text[:eb.cursorBOffset]
}

//InsertRune adds the given rune to the inputbox
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

//String returns the inputbox content as a string
func (eb *InputBox) String() string {
	return string(eb.text)
}

func (eb *InputBox) redrawAll() {
	//	termbox.Clear(coldef, coldef)
	//	w, h := termbox.Size()

	//	midy := h / 2
	//	midx := (w - editBoxWidth) / 2

	eb.Draw(eb.x, eb.y, editBoxWidth, 1)
	termbox.SetCursor(eb.x+eb.CursorX(), eb.y)

	termbox.Flush()
}

//Focus is set on the inputbox, it starts handling terminal events and responding
//to user actions.
func (eb *InputBox) Focus() {
	termbox.SetInputMode(termbox.InputEsc)

	eb.redrawAll()
mainloop:
	for {
		select {
		case ev := <-eb.eventQueue:
			switch ev.Type {
			case termbox.EventKey:
				switch ev.Key {
				case termbox.KeyEnter:
					break mainloop
				case termbox.KeyEsc:
					eb.Delete()
					break mainloop
				case termbox.KeyArrowLeft, termbox.KeyCtrlB:
					eb.MoveCursorOneRuneBackward()
				case termbox.KeyArrowRight, termbox.KeyCtrlF:
					eb.MoveCursorOneRuneForward()
				case termbox.KeyBackspace, termbox.KeyBackspace2:
					eb.DeleteRuneBackward()
				case termbox.KeyDelete, termbox.KeyCtrlD:
					eb.DeleteRuneForward()
				case termbox.KeyTab:
					eb.InsertRune('\t')
				case termbox.KeySpace:
					eb.InsertRune(' ')
				case termbox.KeyCtrlK:
					eb.DeleteTheRestOfTheLine()
				case termbox.KeyHome, termbox.KeyCtrlA:
					eb.MoveCursorToBeginningOfTheLine()
				case termbox.KeyEnd, termbox.KeyCtrlE:
					eb.MoveCursorToEndOfTheLine()
				default:
					if ev.Ch != 0 {
						eb.InsertRune(ev.Ch)
					}
				}
			}
			eb.redrawAll()
		}
	}
	eb.output <- eb.String()
}

//NewInputBox creates an input box, located at position x,y in the screen.
func NewInputBox(x, y int, prompt string, output chan<- string, keyboardQueue chan termbox.Event) *InputBox {
	renderString(x, y, prompt, termbox.ColorWhite, termbox.ColorDefault)
	termbox.Flush()
	return &InputBox{x: x + len(prompt), y: y, output: output, eventQueue: keyboardQueue}
}
