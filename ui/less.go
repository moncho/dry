package ui

import (
	"strings"
	"sync"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/termbox"

	"github.com/moncho/dry/search"
)

const (
	endtext   = "(end)"
	starttext = "(start)"
)

// Less is a View specialization with less-like behavior and characteristics, meaning:
// * The cursor is always shown at the bottom of the screen.
// * Navigation is done using less keybindings.
// * Basic search is supported.
type Less struct {
	*View
	searchResult   *search.Result
	filtering      bool
	following      bool
	refresh        chan struct{}
	screen         *Screen
	renderer       ScreenTextRenderer
	searchHitStyle tcell.Style
	defaultStyle   tcell.Style

	sync.Mutex
}

// NewLess creates a view that partially simulates less.
func NewLess(theme *ColorTheme) *Less {
	sd := ActiveScreen.Dimensions()

	view := NewView("", 0, 0, sd.Width, sd.Height, true, theme)
	view.cursorY = sd.Height - 1 //Last line is at height -1
	less := &Less{
		View:   view,
		screen: ActiveScreen,
	}
	less.renderer = NewRenderer(screenStyledRuneRenderer{ActiveScreen}).WithWidth(sd.Width)
	less.searchHitStyle = mkStyle(termbox.ColorYellow, termbox.Attribute(less.View.theme.Bg))
	less.defaultStyle = mkStyle(termbox.ColorWhite, termbox.Attribute(less.View.theme.Bg))
	return less
}

// Focus sets the view as active, so it starts handling terminal events
// and user actions
func (less *Less) Focus(events <-chan *tcell.EventKey) error {
	refreshChan := make(chan struct{}, 1)

	less.refresh = refreshChan
	less.newLineCallback = func() {
		if less.following {
			//ScrollToBottom refreshes the buffer as well
			less.ScrollToBottom()
		} else {
			less.refreshBuffer()
		}
	}
	inputMode := false

	//This ensures at least one refresh
	less.refreshBuffer()

	go func(inputMode *bool) {

		inputBoxEventChan := make(chan *tcell.EventKey)
		inputBoxOutput := make(chan string, 1)
		defer close(inputBoxOutput)
		defer close(inputBoxEventChan)

		for {
			select {

			case input := <-inputBoxOutput:
				*inputMode = false
				less.search(input)
				less.refreshBuffer()
			case event := <-events:
				if !*inputMode {
					if event.Key() == tcell.KeyEsc {
						less.newLineCallback = func() {}
						close(refreshChan)
						return
					} else if event.Key() == tcell.KeyDown { //cursor down
						less.ScrollDown()
					} else if event.Key() == tcell.KeyUp { // cursor up
						less.ScrollUp()
					} else if event.Key() == tcell.KeyPgDn { //cursor one page down
						less.ScrollPageDown()
					} else if event.Key() == tcell.KeyPgUp { // cursor one page up
						less.ScrollPageUp()
					} else if event.Rune() == 'f' { //toggle follow
						less.flipFollow()
					} else if event.Rune() == 'F' {
						*inputMode = true
						less.filtering = true
						go less.readInput(inputBoxEventChan, inputBoxOutput)
					} else if event.Rune() == 'g' { //to the top of the view
						less.ScrollToTop()
					} else if event.Rune() == 'G' { //to the bottom of the view
						less.ScrollToBottom()
					} else if event.Rune() == 'N' { //to the top of the view
						less.gotoPreviousSearchHit()
					} else if event.Rune() == 'n' { //to the bottom of the view
						less.gotoNextSearchHit()
					} else if event.Rune() == '/' {
						*inputMode = true
						less.filtering = false
						go less.readInput(inputBoxEventChan, inputBoxOutput)
					}
				} else {
					inputBoxEventChan <- event
				}

			}
		}
	}(&inputMode)

	for range less.refresh {
		//If input is being read, refresh events are ignore
		//the only UI changes are happening on the input bar
		//and are done by the InputBox
		if !inputMode {
			less.screen.Clear()
			less.render()
			less.screen.Flush()
		}
	}
	return nil
}

// Search searches in the view buffer for the given pattern
func (less *Less) search(pattern string) error {
	if pattern != "" {
		searchResult, err := search.NewSearch(less.lines, pattern)
		if err == nil {
			less.searchResult = searchResult
			if searchResult.Hits > 0 {
				_, y := less.Position()
				searchResult.InitialLine(y)
				less.gotoNextSearchHit()
			}
		} else {
			return err
		}
	} else {
		less.searchResult = nil
	}
	return nil
}

func (less *Less) readInput(inputBoxEventChan chan *tcell.EventKey, inputBoxOutput chan string) error {
	_, height := less.ViewSize()
	eb := NewInputBox(0, height, ">>> ", inputBoxOutput, inputBoxEventChan, less.theme, less.screen)
	eb.Focus()
	return nil
}

// Render renders the view buffer contents.
func (less *Less) render() {
	less.Lock()
	defer less.Unlock()
	_, maxY := less.renderableArea()
	y := 0

	bufferStart := 0
	if less.bufferY < less.bufferSize() && less.bufferY > 0 {
		bufferStart = less.bufferY
	}
	for _, line := range less.lines[bufferStart:] {

		if y > maxY {
			break
		}
		less.renderLine(0, y, string(line))
		y++
	}

	less.renderStatusLine()
	less.drawCursor()
}

func (less *Less) flipFollow() {
	less.following = !less.following
	if less.following {
		less.ScrollToBottom()
	} else {
		less.refreshBuffer()
	}
}

// ScrollDown moves the cursor down one line
func (less *Less) ScrollDown() {
	less.scrollDown(1)
}

// ScrollUp moves the cursor up one line
func (less *Less) ScrollUp() {
	less.scrollUp(1)
}

// ScrollPageDown moves the buffer position down by the length of the screen,
// at the end of buffer it also moves the cursor position to the bottom
// of the screen
func (less *Less) ScrollPageDown() {
	_, height := less.renderableArea()
	less.scrollDown(height)

}

// ScrollPageUp moves the buffer position up by the length of the screen,
// at the beginning of buffer it also moves the cursor position to the beginning
// of the screen
func (less *Less) ScrollPageUp() {
	_, height := less.renderableArea()
	less.scrollUp(height)

}

// ScrollToBottom moves the cursor to the bottom of the view buffer
func (less *Less) ScrollToBottom() {
	less.bufferY = less.bufferSize() - less.y1
	less.refreshBuffer()

}

// ScrollToTop moves the cursor to the top of the view buffer
func (less *Less) ScrollToTop() {
	less.bufferY = 0
	less.refreshBuffer()
}

func (less *Less) atTheStartOfBuffer() bool {
	_, y := less.Position()
	return y == 0
}

func (less *Less) atTheEndOfBuffer() bool {
	viewLength := less.bufferSize()
	_, y := less.Position()
	_, height := less.ViewSize()
	return y+height >= viewLength-1
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
	less.refreshBuffer()
}
func (less *Less) gotoNextSearchHit() {
	sr := less.searchResult
	if sr != nil {
		x, _ := less.Position()
		if newY, err := sr.NextLine(); err == nil {
			less.setPosition(x, newY)
		}
	}
	less.refreshBuffer()
}

func (less *Less) refreshBuffer() {
	//Non blocking send. Since the refresh channel is buffered, losing
	//refresh messages because of a full buffer should not be a problem
	//since there is already a refresh message waiting to be processed.
	select {
	case less.refresh <- struct{}{}:
	default:
	}
}

// renderableArea return the part of the view size available for rendering.
func (less *Less) renderableArea() (int, int) {
	maxX, maxY := less.ViewSize()
	return maxX, maxY - 1
}

func (less *Less) renderLine(x int, y int, line string) (int, error) {
	var lines = 1
	maxWidth, _ := less.renderableArea()
	if less.searchResult != nil {
		//If markup support is active then it might happen that tags are present in the line
		//but since we are searching, markups are ignored and coloring output is
		//decided here.
		if strings.Contains(line, less.searchResult.Pattern) {
			if less.markup != nil {
				var builder strings.Builder
				for _, token := range Tokenize(line, SupportedTags) {
					if less.markup.IsTag(token) {
						continue
					}
					builder.WriteString(token)
				}
				line = builder.String()
			}
			lines = less.renderer.On(x, y).WithStyle(less.searchHitStyle).WithWidth(maxWidth).Render(line)

		} else if !less.filtering {
			return less.View.renderLine(x, y, line)
		}

	} else {
		return less.View.renderLine(x, y, line)
	}
	return lines, nil
}

// scrollDown moves the buffer position down by the given number of lines
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
	less.refreshBuffer()

}

// scrollUp moves the buffer position up by the given number of lines
func (less *Less) scrollUp(lines int) {
	ox, bufferY := less.Position()
	if bufferY-lines >= 0 {
		less.setPosition(ox, bufferY-lines)
	} else {
		less.setPosition(ox, 0)
	}
	less.refreshBuffer()
}

func (less *Less) renderStatusLine() {
	maxWidth, maxLength := less.ViewSize()
	var cursorX = 1
	status := less.statusLine()
	if less.atTheEndOfBuffer() {
		cursorX = len(endtext)
	} else if less.atTheStartOfBuffer() {
		cursorX = len(starttext)
	}
	less.renderer.On(0, maxLength).WithStyle(
		less.defaultStyle).WithWidth(maxWidth).Render(status)
	less.cursorX = cursorX
}

func (less *Less) statusLine() string {
	maxWidth, _ := less.ViewSize()

	var start string
	switch {
	case less.atTheStartOfBuffer():
		start = starttext
	case less.atTheEndOfBuffer():
		start = endtext
	default:
		start = ":"
	}

	var end string
	if less.filtering && less.searchResult != nil {
		end = strings.Join([]string{less.searchResult.String(), "Filter: On"}, " ")
	} else {
		end = "Filter: Off"
	}

	if less.following {
		end += " Follow: On"
	} else {
		end += " Follow: Off"
	}

	return strings.Join(
		[]string{start, end},
		strings.Repeat(" ", maxWidth-len(start)-len(end)))
}

func (less *Less) drawCursor() {
	less.screen.ShowCursor(less.Cursor())
}
