package ui

import (
	"strings"

	"github.com/moncho/dry/search"
	"github.com/nsf/termbox-go"
)

const (
	endtext   = "(end)"
	starttext = "(start)"
)

//Less is a View with less-like behaviour and characteristics, meaning:
// * The cursor is always shown at the bottom of the screen
type Less struct {
	*View
	searchResult *search.Result
}

//NewLess creates a view that partially simulates less.
func NewLess() *Less {
	width, height := termbox.Size()
	view := &View{
		name:       "",
		x1:         width,
		y1:         height,
		cursorX:    0,
		cursorY:    height - 1, //Last line is at height -1
		showCursor: true,
	}
	return &Less{
		view, nil,
	}
}

//Focus sets the view as active, it starts handling terminal events
//and user actions
func (less *Less) Focus(keyboardQueue <-chan termbox.Event) error {
	clear()
	if err := less.Render(); err != nil {
		return err
	}
	termbox.Flush()
	inputMode := false
	inputBoxEventChan := make(chan termbox.Event)
	inputBoxOuput := make(chan string, 1)
	defer close(inputBoxOuput)
	defer close(inputBoxEventChan)
loop:
	for {
		select {
		case input := <-inputBoxOuput:
			inputMode = false
			less.Search(input)
			clear()
			if err := less.Render(); err != nil {
				return err
			}
			termbox.Flush()
		case event := <-keyboardQueue:
			switch event.Type {
			case termbox.EventKey:
				if !inputMode {
					if event.Key == termbox.KeyEsc {
						break loop
					} else if event.Key == termbox.KeyArrowDown { //cursor down
						less.ScrollDown()
					} else if event.Key == termbox.KeyArrowUp { // cursor up
						less.ScrollUp()
					} else if event.Key == termbox.KeyPgdn { //cursor one page down
						less.ScrollPageDown()
					} else if event.Key == termbox.KeyPgup { // cursor one page up
						less.ScrollPageUp()
					} else if event.Ch == 'N' { //to the top of the view
						less.gotoPreviousSearchHit()
					} else if event.Ch == 'n' { //to the bottom of the view
						less.gotoNextSearchHit()
					} else if event.Ch == 'g' { //to the top of the view
						less.ScrollToTop()
					} else if event.Ch == 'G' { //to the bottom of the view
						less.ScrollToBottom()
					} else if event.Ch == '/' {
						inputMode = true
						less.tainted = false
						go less.readInput(inputBoxEventChan, inputBoxOuput)
					}

					if less.tainted {
						clear()
						if err := less.Render(); err != nil {
							return err
						}
						termbox.Flush()
					}
				} else {
					inputBoxEventChan <- event
				}
			}
		}
	}
	return nil
}

//Search searchs in the view buffer the given pattern
func (less *Less) Search(pattern string) error {
	if pattern != "" {
		less.tainted = true
		searchResult, err := search.NewSearch(less.lines, pattern)
		if err == nil {
			less.searchResult = searchResult
			_, y := less.Position()
			searchResult.InitialLine(y)
		} else {
			return err
		}
	} else {
		less.searchResult = nil
	}
	return nil
}

func (less *Less) readInput(inputBoxEventChan chan termbox.Event, inputBoxOuput chan string) error {
	_, height := less.ViewSize()
	eb := NewInputBox(0, height, "/", inputBoxOuput, inputBoxEventChan)
	eb.Focus()
	return nil
}

// Render renders the view buffer contents.
func (less *Less) Render() error {
	_, maxY := less.renderSize()

	//less.prepareViewForRender()

	y := 0
	for i, vline := range less.lines {
		if i < less.bufferY {
			continue
		}
		if y > maxY {
			break
		}
		less.renderLine(0, y, string(vline))
		y++
	}

	less.renderMessage()
	less.drawCursor()
	return nil
}

//ScrollDown moves the cursor down one line
func (less *Less) ScrollDown() {
	less.scrollDown(1)
}

//ScrollUp moves the cursor up one line
func (less *Less) ScrollUp() {
	less.scrollUp(1)
}

//ScrollPageDown moves the buffer position down by the length of the screen,
//at the end of buffer it also moves the cursor position to the bottom
//of the screen
func (less *Less) ScrollPageDown() {
	_, height := less.ViewSize()
	less.scrollDown(height)
}

//ScrollPageUp moves the buffer position up by the length of the screen,
//at the beginning of buffer it also moves the cursor position to the beginning
//of the screen
func (less *Less) ScrollPageUp() {
	_, height := less.ViewSize()
	less.scrollUp(height)
}

//ScrollToBottom moves the cursor to the bottom of the view buffer
func (less *Less) ScrollToBottom() {
	less.bufferY = less.bufferSize() - less.y1
	less.tainted = true

}

//ScrollToTop moves the cursor to the top of the view buffer
func (less *Less) ScrollToTop() {
	less.bufferY = 0
	less.tainted = true

}

func (less *Less) atTheStartOfBuffer() bool {
	_, y := less.Position()
	if y == 0 {
		return true
	}
	return false
}

func (less *Less) atTheEndOfBuffer() bool {
	viewLength := less.bufferSize()
	_, y := less.Position()
	_, height := less.ViewSize()
	if y+height >= viewLength-1 {
		return true
	}
	return false
}

func (less *Less) bufferSize() int {
	return len(less.lines)
}

func (less *Less) gotoPreviousSearchHit() {
	sr := less.searchResult
	if sr != nil {
		x, _ := less.Position()
		if newy, err := sr.PreviousLine(); err == nil {
			less.setPosition(x, newy)
		}
	}
}
func (less *Less) gotoNextSearchHit() {
	sr := less.searchResult
	if sr != nil {
		x, _ := less.Position()
		if newy, err := sr.NextLine(); err == nil {
			less.setPosition(x, newy)
		}
	}
}

//renderSize return the part of the view size available for rendering.
func (less *Less) renderSize() (int, int) {
	maxX, maxY := less.ViewSize()
	return maxX, maxY - 1
}

func (less *Less) renderLine(x int, y int, line string) error {
	var fg, bg = termbox.ColorDefault, termbox.ColorDefault
	for _, token := range strings.Fields(line) {
		if less.searchResult != nil && strings.Contains(token, less.searchResult.Pattern) {
			fg = termbox.ColorYellow
		}
		renderString(x, y, line, fg, bg)
	}
	return nil
}

//scrollDown moves the buffer position down by the given number of lines
func (less *Less) scrollDown(lines int) {
	_, height := less.ViewSize()
	viewLength := less.bufferSize()

	posX, posY := less.Position()
	//This is a down as scrolling can go
	maxY := viewLength - height
	if posY+lines < maxY {
		newOy := posY + lines
		if newOy >= viewLength {
			less.setPosition(posX, viewLength-height)
		} else {
			less.setPosition(posX, newOy)
		}
	} else {

		less.ScrollToBottom()
	}
	less.tainted = true
}

//scrollUp moves the buffer position up by the given number of lines
func (less *Less) scrollUp(lines int) {
	ox, bufferY := less.Position()
	if bufferY-lines >= 0 {
		less.setPosition(ox, bufferY-lines)
	} else {
		less.setPosition(ox, 0)
	}
	less.tainted = true
}

func (less *Less) renderMessage() {
	_, maxY := less.ViewSize()
	var cursorX = 1
	switch {
	case less.searchResult != nil:
		{
			renderString(0, maxY, less.searchResult.String(), termbox.ColorWhite, termbox.ColorDefault)
			cursorX = len(less.searchResult.String())
		}
	case !less.atTheEndOfBuffer() && !less.atTheStartOfBuffer():
		termbox.SetCell(0, maxY, ':', termbox.ColorDefault, termbox.ColorDefault)
	case less.atTheStartOfBuffer():
		renderString(0, maxY, starttext, termbox.ColorWhite, termbox.ColorDefault)
		cursorX = len(starttext)
	default:
		{
			renderString(0, maxY, endtext, termbox.ColorWhite, termbox.ColorDefault)
			cursorX = len(endtext)
		}
	}
	less.cursorX = cursorX
}

func (less *Less) drawCursor() {
	x, y := less.Cursor()

	termbox.SetCursor(x, y)
}

func clear() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	termbox.Flush()
}
