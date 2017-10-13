package termui

import (
	"errors"

	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

// TextInput is a widget for text input
type TextInput struct {
	termui.Block
	input         []rune
	cursorX       int
	cursorY       int
	cursorLinePos int
	isCapturing   bool
	escaped       bool //tracks if the input process was finished (i.e. user pressed Enter) or exited (i.e. user pressed Esc)
	TextFgColor   termui.Attribute
	TextBgColor   termui.Attribute
	TextBuilder   termui.TextBuilder
}

//NewTextInput creates a new TextInput
func NewTextInput(s string) *TextInput {
	textInput := &TextInput{
		Block:         *termui.NewBlock(),
		TextBuilder:   termui.NewMarkdownTxBuilder(),
		cursorLinePos: 0,
	}

	if s != "" {
		textInput.setText(s)
	}

	return textInput
}

//OnFocus starts handling events sent to the given channel. It is a
//blocking call, to return from it, either close the channel or
//sent a closing event (i.e. KeyEnter on single line mode, KeyEsc on any mode).
func (i *TextInput) OnFocus(event ui.EventSource) error {
	if i.isCapturing {
		return errors.New("This text input is already capturing events")
	}
	i.isCapturing = true
	i.escaped = false
	var err error
mainloop:
	for ev := range event.Events {

		switch ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEnter:
				err = event.EventHandledCallback(ev)
				break mainloop
			case termbox.KeyEsc:
				err = event.EventHandledCallback(ev)
				i.escaped = true
				break mainloop
			case termbox.KeyArrowLeft, termbox.KeyCtrlB:
				i.moveLeft()
			case termbox.KeyArrowRight, termbox.KeyCtrlF:
				i.moveRight()
			case termbox.KeyBackspace, termbox.KeyBackspace2:
				i.backspace()
			case termbox.KeyDelete, termbox.KeyCtrlD:
				i.deleteRuneForward()
			case termbox.KeyTab:
				i.addRune('\t')
			case termbox.KeySpace:
				i.addRune(' ')
			case termbox.KeyCtrlK:
				i.deleteTheRestOfTheLine()
			case termbox.KeyHome, termbox.KeyCtrlA:
				i.moveCursorToBeginningOfTheLine()
			case termbox.KeyEnd, termbox.KeyCtrlE:
				i.moveCursorToEndOfTheLine()

			default:
				if ev.Ch != 0 {
					i.addRune(ev.Ch)
				}
			}
		}
		err = event.EventHandledCallback(ev)
		if err != nil {
			break mainloop
		}
	}
	termbox.HideCursor()
	i.isCapturing = false
	return err
}

// Text returns the text of the input field as a string
func (i *TextInput) Text() (string, bool) {
	if len(i.input) == 0 {
		return "", i.escaped
	}

	return string(i.input), i.escaped
}

func (i *TextInput) setText(text string) {
	i.input = []rune(text)
	i.cursorLinePos = len(i.input)
}

func (i *TextInput) backspace() {
	runeCount := len(i.input)
	if runeCount == 0 {
		return
	}

	// at the beginning
	if i.cursorLinePos == 0 {
		return
	}

	//at the end of a line
	if i.cursorLinePos == runeCount-1 {
		i.input = i.input[:runeCount-1]
		i.cursorLinePos--
		return
	}

	// at the middle of a line
	i.input = append(i.input[:i.cursorLinePos-1], i.input[i.cursorLinePos:]...)
	i.cursorLinePos--
}

func (i *TextInput) addRune(r rune) {
	// cursor is not at the beggining or the end of the input
	if i.cursorLinePos > 0 && i.cursorLinePos < len(i.input) {
		before := i.input[:i.cursorLinePos]
		after := i.input[i.cursorLinePos:]
		i.input = append(append(before, r), after...)
	} else {
		i.input = append(i.input, r)
	}
	i.cursorLinePos++
}

func (i *TextInput) deleteRuneForward() {
	runeCount := len(i.input)
	if runeCount == 0 {
		return
	}

	if i.cursorLinePos < runeCount-1 {
		i.input = append(i.input[:i.cursorLinePos], i.input[i.cursorLinePos+1:]...)
	}
}

func (i *TextInput) deleteTheRestOfTheLine() {
	runeCount := len(i.input)
	if runeCount == 0 {
		return
	}

	if i.cursorLinePos < runeCount-1 {
		i.input = i.input[:i.cursorLinePos]
	}
}
func (i *TextInput) moveCursorToBeginningOfTheLine() {
	i.cursorLinePos = 0
}

func (i *TextInput) moveCursorToEndOfTheLine() {
	i.cursorLinePos = len(i.input)
}
func (i *TextInput) moveLeft() {
	if i.cursorLinePos == 0 {
		return
	}
	i.cursorLinePos--
}

func (i *TextInput) moveRight() {
	if i.cursorLinePos >= len(i.input) {
		return
	}

	i.cursorLinePos++
}

//Buffer returns the content of this widget as a termui.Buffer
func (i *TextInput) Buffer() termui.Buffer {
	buffer := i.Block.Buffer()
	innerArea := i.InnerBounds()
	text := string(i.input)

	fg, bg := i.TextFgColor, i.TextBgColor
	cells := i.TextBuilder.Build(text, fg, bg)
	textXOffset := 0
	if innerArea.Dx() < len(cells) {
		textXOffset = len(cells) - innerArea.Dx()
	}
	x := 0
	for _, cell := range cells[textXOffset:] {
		w := cell.Width()
		buffer.Set(innerArea.Min.X+x, innerArea.Min.Y, cell)
		x += w
	}

	cursorXOffset := i.X
	if i.BorderLeft {
		cursorXOffset++
	}

	cursorYOffset := i.Y
	if i.BorderTop {
		cursorYOffset++
	}

	if i.isCapturing {
		i.cursorX = i.cursorLinePos + cursorXOffset - textXOffset
		i.cursorY = cursorYOffset
		termbox.SetCursor(i.cursorX, i.cursorY)
	}

	return buffer
}
