package ui

import (
	"strings"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

// ExpiringMessageWidget shows some text for an amount time then clears itself
type ExpiringMessageWidget struct {
	y           int
	screenWidth int
	clearTimer  *time.Timer
	markup      *Markup

	sync.RWMutex
	message string
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
	s.stopTimer()
	s.Unlock()

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
	if clearDelay == 0 {
		return
	}
	s.clearTimer = time.AfterFunc(clearDelay, func() {
		s.Lock()
		s.message = ""
		s.Unlock()
		clearMessage := strings.Repeat(" ", len(msg))
		renderString(0, s.y, s.screenWidth, clearMessage, termbox.Attribute(s.markup.theme.Fg), termbox.Attribute(s.markup.theme.Bg))
	})

}

//Render renders the status message
func (s *ExpiringMessageWidget) Render() {
	if s.message == "" {
		return
	}
	s.RLock()
	defer s.RUnlock()
	w, _ := termbox.Size()
	renderLineWithMarkup(0, s.y, w, s.message, s.markup)
}
