package ui

import (
	"strings"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

// ExpiringMessageWidget shows some text for an amount time then clears itself
type ExpiringMessageWidget struct {
	sync.Mutex
	y           int
	screenWidth int
	message     string
	clearTimer  *time.Timer
	markup      *Markup
}

// NewExpiringMessageWidget creates a new ExpiringMessageWidget struct
func NewExpiringMessageWidget(y, screenWidth int, theme *ColorTheme) *ExpiringMessageWidget {
	return &ExpiringMessageWidget{
		y:           y,
		screenWidth: screenWidth,
		clearTimer:  nil,
		markup:      NewMarkup(theme),
	}

}

//Pause pauses this widget from showing any output, setting a new status
//message will activate it again
func (s *ExpiringMessageWidget) Pause() {
	s.Lock()
	defer s.Unlock()
	s.stopTimer()
}

func (s *ExpiringMessageWidget) stopTimer() {
	if s.clearTimer != nil {
		s.clearTimer.Stop()
		s.clearTimer = nil
	}
}

// Message sets the message to show for the given duration
func (s *ExpiringMessageWidget) Message(msg string, clearDelay time.Duration) {
	s.Lock()
	defer s.Unlock()
	s.stopTimer()
	s.message = msg
	if clearDelay != 0 {
		s.clearTimer = time.AfterFunc(clearDelay, func() {
			clearMessage := strings.Repeat(" ", len(msg))
			s.message = ""
			renderString(0, s.y, s.screenWidth, clearMessage, termbox.Attribute(s.markup.theme.Fg), termbox.Attribute(s.markup.theme.Bg))
		})
	}
}

//Render renders the status message
func (s *ExpiringMessageWidget) Render() {
	s.Lock()
	defer s.Unlock()
	if s.message != "" {
		w, _ := termbox.Size()
		renderLineWithMarkup(0, s.y, w, s.message, s.markup)
	}
}
