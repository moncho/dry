package ui

import (
	"strings"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

// StatusBar draws the status message bar
type StatusBar struct {
	screenPos   int
	lastMessage string
	clearTimer  *time.Timer
	barMutex    sync.Locker
	markup      *Markup
}

// NewStatusBar creates a new StatusBar struct
func NewStatusBar(screenPos int) *StatusBar {
	markup := NewMarkup()
	return &StatusBar{
		screenPos:  screenPos,
		clearTimer: nil,
		barMutex:   &sync.Mutex{},
		markup:     markup,
	}

}

func (s *StatusBar) stopTimer() {
	s.barMutex.Lock()
	defer s.barMutex.Unlock()
	if t := s.clearTimer; t != nil {
		t.Stop()
		s.clearTimer = nil
	}
}

func (s *StatusBar) setClearTimer(t *time.Timer) {
	s.barMutex.Lock()
	defer s.barMutex.Unlock()
	s.clearTimer = t
}

// StatusMessage sets a new status message for the given duration
func (s *StatusBar) StatusMessage(msg string, clearDelay time.Duration) {
	s.stopTimer()
	s.lastMessage = msg
	//set a timer to clear the status
	if clearDelay != 0 {
		s.setClearTimer(time.AfterFunc(clearDelay, func() {
			clearMessage := strings.Repeat(" ", len(msg))
			s.lastMessage = ""
			renderString(0, s.screenPos, string(clearMessage), termbox.ColorDefault, termbox.ColorDefault)
			termbox.Flush()
		}))
	}
}

//Render renders the status message
func (s *StatusBar) Render() {
	s.barMutex.Lock()
	defer s.barMutex.Unlock()
	if s.lastMessage != "" {
		w, _ := termbox.Size()
		renderLineWithMarkup(0, s.screenPos, w, s.lastMessage, s.markup)
	}
}
