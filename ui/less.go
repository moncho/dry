package ui

import (
	"strings"
	"time"

	"github.com/moncho/dry/search"
	"github.com/nsf/termbox-go"
)

const (
	endtext   = "(end)"
	starttext = "(start)"
)

//Less is a View specilization with less-like behaviour and characteristics, meaning:
// * The cursor is always shown at the bottom of the screen.
// * Navigation is done using less keybindings.
// * Basic search is supported.
type Less struct {
	*View
	searchResult *search.Result
	filtering    bool
}

//NewLess creates a view that partially simulates less.
func NewLess() *Less {
	width, height := termbox.Size()
	view := NewView("", 0, 0, width, height, true)
	view.cursorY = height - 1 //Last line is at height -1

	return &Less{
		view, nil, false,
	}
}

//Focus sets the view as active, so it starts handling terminal events
//and user actions
func (less *Less) Focus(events <-chan termbox.Event) error {
	inputMode := false
	inputBoxEventChan := make(chan termbox.Event)
	inputBoxOuput := make(chan string, 1)
	refreshTimer := time.NewTicker(500 * time.Millisecond)
	stop := make(chan struct{})

	//the first render is done when some content is added to the buffer
	go func() {
		for {
			select {
			case <-refreshTimer.C:
				if less.bufferSize() > 0 {
					less.tainted = false
					less.Render()
					termbox.Flush()
					return
				}
			case <-stop:
				return
			}
		}
	}()

	defer close(inputBoxOuput)
	defer close(inputBoxEventChan)
	defer close(stop)
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
		case event := <-events:
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
						less.filtering = false
						go less.readInput(inputBoxEventChan, inputBoxOuput)
					} else if event.Ch == 'f' {
						inputMode = true
						less.tainted = false
						less.filtering = true
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

//Search searchs in the view buffer for the given pattern
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
	eb := NewInputBox(0, height, ">>>", inputBoxOuput, inputBoxEventChan)
	eb.Focus()
	return nil
}

// Render renders the view buffer contents.
func (less *Less) Render() error {
	_, maxY := less.renderSize()
	y := 0

	bufferStart := 0
	if less.bufferY < less.bufferSize() && less.bufferY > 0 {
		bufferStart = less.bufferY
	}
	for _, vline := range less.lines[bufferStart:] {

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

func (less *Less) renderLine(x int, y int, line string) (int, error) {
	var lines = 1
	maxWidth, _ := less.renderSize()
	if less.searchResult != nil {
		//If markup support is active then it might happen that tags are present in the line
		//but since we are searching, markups are ignored and coloring output is
		//decided here.
		if strings.Contains(line, less.searchResult.Pattern) {
			if less.markup != nil {
				start, column := 0, 0
				for _, token := range Tokenize(line, supportedTags) {
					if less.markup.IsTag(token) {

						continue
					}
					// Here comes the actual text: display it one character at a time.
					for _, char := range token {
						start = x + column
						column++
						termbox.SetCell(start, y, char, termbox.ColorYellow, termbox.ColorDefault)
					}
				}
			} else {
				_, lines = renderString(x, y, maxWidth, line, termbox.ColorYellow, termbox.ColorDefault)
			}
		} else if !less.filtering {
			return less.View.renderLine(x, y, line)
		}

	} else {
		return less.View.renderLine(x, y, line)
	}
	return lines, nil
}

//scrollDown moves the buffer position down by the given number of lines
func (less *Less) scrollDown(lines int) {
	_, height := less.ViewSize()
	viewLength := less.bufferSize()

	posX, posY := less.Position()
	//This is as down as scrolling can go
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
	maxWidth, maxLength := less.ViewSize()
	var cursorX = 1
	switch {
	case less.searchResult != nil:
		{
			renderString(0, maxLength, maxWidth, less.searchResult.String(), termbox.ColorWhite, termbox.ColorDefault)
			cursorX = len(less.searchResult.String())
		}
	case !less.atTheEndOfBuffer() && !less.atTheStartOfBuffer():
		termbox.SetCell(0, maxLength, ':', termbox.ColorDefault, termbox.ColorDefault)
	case less.atTheStartOfBuffer():
		renderString(0, maxLength, maxWidth, starttext, termbox.ColorWhite, termbox.ColorDefault)
		cursorX = len(starttext)
	default:
		{
			renderString(0, maxLength, maxWidth, endtext, termbox.ColorWhite, termbox.ColorDefault)
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
}
