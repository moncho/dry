package termui

import (
	"strconv"
	"strings"

	"github.com/gizak/termui"
	"github.com/nsf/termbox-go"
)

const (
	newLine = "\n"
)

// TextInput is a widget for text input
type TextInput struct {
	termui.Block
	TextFgColor termui.Attribute
	TextBgColor termui.Attribute
	isCapturing bool
	TextBuilder termui.TextBuilder

	ShowLineNo bool
	cursorX    int
	cursorY    int

	cursorLineIndex int
	cursorLinePos   int
	isMultiLine     bool
	lines           []string
}

//NewTextInput creates a new TextInput
func NewTextInput(s string, isMultiLine bool) *TextInput {
	textInput := &TextInput{
		Block:           *termui.NewBlock(),
		TextBuilder:     termui.NewMarkdownTxBuilder(),
		isMultiLine:     isMultiLine,
		ShowLineNo:      false,
		cursorLineIndex: 0,
		cursorLinePos:   0,
	}

	if s != "" {
		textInput.setText(s)
	}

	return textInput
}

//Focus starts handling events sent to the given channel. It is a
//blocking call, to return from it, either close the channel or
//sent a closing event (i.e. KeyEnter on single line mode, KeyEsc on any mode).
func (i *TextInput) Focus(events <-chan termbox.Event) error {
	i.isCapturing = true
	//	termbox.SetInputMode(termbox.InputEsc)

mainloop:
	for ev := range events {

		switch ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEnter:
				if i.isMultiLine {
					i.addString("\n")
				} else {
					break mainloop
				}
			case termbox.KeyEsc:
				break mainloop
			case termbox.KeyArrowLeft, termbox.KeyCtrlB:
				i.moveLeft()
			case termbox.KeyArrowRight, termbox.KeyCtrlF:
				i.moveRight()
			case termbox.KeyBackspace, termbox.KeyBackspace2:
				i.backspace()
			case termbox.KeyDelete, termbox.KeyCtrlD:
				//TODO i.deleteRuneForward()
			case termbox.KeyTab:
				i.addString("\t")
			case termbox.KeySpace:
				i.addString(" ")
			case termbox.KeyCtrlK:
				//TODO i.deleteTheRestOfTheLine()
			case termbox.KeyHome, termbox.KeyCtrlA:
				//TODO i.MoveCursorToBeginningOfTheLine()
			case termbox.KeyEnd, termbox.KeyCtrlE:
				//TODO i.MoveCursorToEndOfTheLine()
			case termbox.KeyArrowUp:
				i.moveUp()
			case termbox.KeyArrowDown:
				i.moveDown()

			default:
				if ev.Ch != 0 {
					i.addString(string(ev.Ch))
				}
			}
		}
	}
	return nil
}

// Text returns the text of the input field as a string
func (i *TextInput) Text() string {
	if len(i.lines) == 0 {
		return ""
	}

	if i.isMultiLine {
		return strings.Join(i.lines, newLine)
	}
	return i.lines[0]
}

func (i *TextInput) setText(text string) {
	i.lines = strings.Split(text, newLine)
}

// Lines returns the slice of strings with the content of the input field.
func (i *TextInput) Lines() []string {
	return i.lines
}

func (i *TextInput) backspace() {
	curLine := i.lines[i.cursorLineIndex]
	// at the beginning of the buffer, nothing to do
	if len(curLine) == 0 && i.cursorLineIndex == 0 {
		return
	}

	// at the beginning of a line somewhere in the buffer
	if i.cursorLinePos == 0 {
		prevLine := i.lines[i.cursorLineIndex-1]
		// remove the newline character from the prevline
		prevLine = prevLine[:len(curLine)-1] + curLine
		i.lines = append(i.lines[:i.cursorLineIndex], i.lines[i.cursorLineIndex+1:]...)
		i.cursorLineIndex--
		i.cursorLinePos = len(prevLine) - 1
		return
	}

	//at the end of a line
	if i.cursorLinePos == len(curLine)-1 {
		i.lines[i.cursorLineIndex] = curLine[:len(curLine)-1]
		i.cursorLinePos--
		return
	}

	// at the middle of a line
	i.lines[i.cursorLineIndex] = curLine[:i.cursorLinePos-1] + curLine[i.cursorLinePos:]
	i.cursorLinePos--
}

func (i *TextInput) addString(s string) {
	if len(i.lines) > 0 {
		if s == newLine {
			// special case when we go back to the beginning of a buffer with multiple lines, prepend a new line
			if i.cursorLineIndex == 0 && len(i.lines) > 1 {
				i.lines = append([]string{""}, i.lines...)

				// this case handles newlines at the end of the file or in the middle of the file
			} else {
				newString := ""

				// if we are inserting a newline in a populated line then set what goes into the new line
				// and what stays in the current line
				if i.cursorLinePos < len(i.lines[i.cursorLineIndex]) {
					newString = i.lines[i.cursorLineIndex][i.cursorLinePos:]
					i.lines[i.cursorLineIndex] = i.lines[i.cursorLineIndex][:i.cursorLinePos]
				}

				// append a newline in the current position with the content we computed in the previous if statement
				i.lines = append(
					i.lines[:i.cursorLineIndex+1],
					append(
						[]string{newString},
						i.lines[i.cursorLineIndex+1:]...,
					)...,
				)
			}
			// increment the line index, reset the cursor to the beginning and return to skip the next step
			i.cursorLineIndex++
			i.cursorLinePos = 0
			return
		}

		// cursor is at the end of the line
		if i.cursorLinePos == len(i.lines[i.cursorLineIndex]) {
			//i.debugMessage ="end"
			i.lines[i.cursorLineIndex] += s

			// cursor at the beginning of the line
		} else if i.cursorLinePos == 0 {
			i.lines[i.cursorLineIndex] = s + i.lines[i.cursorLineIndex]

			// cursor in the middle of the line
		} else {
			before := i.lines[i.cursorLineIndex][:i.cursorLinePos]
			after := i.lines[i.cursorLineIndex][i.cursorLinePos:]
			i.lines[i.cursorLineIndex] = before + s + after

		}
		i.cursorLinePos += len(s)

	} else {
		i.lines = append(i.lines, s)
		i.cursorLinePos += len(s)
	}
}

func (i *TextInput) moveUp() {
	// if we are already on the first line then just move the cursor to the beginning
	if i.cursorLineIndex == 0 {
		i.cursorLinePos = 0
		return
	}

	// The previous line is just as long, we can move to the same position in the line
	prevLine := i.lines[i.cursorLineIndex-1]
	if len(prevLine) >= i.cursorLinePos {
		i.cursorLineIndex--
	} else {
		// otherwise we move the cursor to the end of the previous line
		i.cursorLineIndex--
		i.cursorLinePos = len(prevLine) - 1
	}
}

func (i *TextInput) moveDown() {
	// we are already on the last line, we just need to move the position to the end of the line
	if i.cursorLineIndex == len(i.lines)-1 {
		i.cursorLinePos = len(i.lines[i.cursorLineIndex])
		return
	}

	// check if the cursor can move to the same position in the next line, otherwise move it to the end
	nextLine := i.lines[i.cursorLineIndex+1]
	if len(nextLine) >= i.cursorLinePos {
		i.cursorLineIndex++
	} else {
		i.cursorLineIndex++
		i.cursorLinePos = len(nextLine) - 1
	}
}

func (i *TextInput) moveLeft() {
	// if we are at the beginning of the line move the cursor to the previous line
	if i.cursorLinePos == 0 {
		origLine := i.cursorLineIndex
		i.moveUp()
		if origLine > 0 {
			i.cursorLinePos = len(i.lines[i.cursorLineIndex])
		}
		return
	}

	i.cursorLinePos--
}

func (i *TextInput) moveRight() {
	// if at the end of the line move to the next
	if i.cursorLinePos >= len(i.lines[i.cursorLineIndex]) {
		origLine := i.cursorLineIndex
		i.moveDown()
		if origLine < len(i.lines)-1 {
			i.cursorLinePos = 0
		}
		return
	}

	i.cursorLinePos++
}

//Buffer returns the content of this widget as a termui.Buffer
func (i *TextInput) Buffer() termui.Buffer {
	buffer := i.Block.Buffer()

	// offset used to display the line numbers
	textXOffset := 0

	bufferLines := i.lines
	firstLine := 0
	innerArea := i.InnerBounds()
	lastLine := innerArea.Dy()
	if i.isMultiLine {
		if i.cursorLineIndex >= lastLine {
			firstLine += i.cursorLineIndex - lastLine + 1
			lastLine += i.cursorLineIndex - lastLine + 1
		}

		if len(i.lines) < lastLine {
			bufferLines = i.lines[firstLine:]
		} else {
			bufferLines = i.lines[firstLine:lastLine]
		}
	}

	text := strings.Join(bufferLines, newLine)

	// when showing line numbers, if the last line is empty then we add an extra space to ensure line numbers are displayed
	if i.ShowLineNo && len(bufferLines) > 0 && bufferLines[len(bufferLines)-1] == "" {
		text += " "
	}

	fg, bg := i.TextFgColor, i.TextBgColor
	cs := i.TextBuilder.Build(text, fg, bg)
	y, x, n := 0, 0, 0
	lineNoCnt := 1

	for n < len(cs) {
		w := cs[n].Width()

		if x == 0 && i.ShowLineNo {
			curLineNoString := " " + strconv.Itoa(lineNoCnt) +
				strings.Join(make([]string, textXOffset-len(strconv.Itoa(lineNoCnt))-1), " ")
			curLineNoRunes := i.TextBuilder.Build(curLineNoString, fg, bg)
			for lineNo := 0; lineNo < len(curLineNoRunes); lineNo++ {
				buffer.Set(innerArea.Min.X+x+lineNo, innerArea.Min.Y+y, curLineNoRunes[lineNo])
			}
			lineNoCnt++
		}

		if cs[n].Ch == '\n' {
			y++
			n++
			x = 0
			continue
		}
		buffer.Set(innerArea.Min.X+x+textXOffset, innerArea.Min.Y+y, cs[n])

		n++
		x += w
	}

	cursorXOffset := i.X + textXOffset
	if i.BorderLeft {
		cursorXOffset++
	}

	cursorYOffset := i.Y
	if i.BorderTop {
		cursorYOffset++
	}
	if lastLine > innerArea.Dy() {
		cursorYOffset += innerArea.Dy() - 1
	} else {
		cursorYOffset += i.cursorLineIndex
	}
	if i.isCapturing {
		i.cursorX = i.cursorLinePos + cursorXOffset
		i.cursorY = cursorYOffset
		termbox.SetCursor(i.cursorLinePos+cursorXOffset, cursorYOffset)
	}

	return buffer
}
