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
	timerMutex  sync.Locker
	markup      *Markup
}

// NewStatusBar creates a new StatusBar struct
func NewStatusBar(screenPos int) *StatusBar {
	return &StatusBar{
		screenPos:  screenPos,
		clearTimer: nil,
		timerMutex: &sync.Mutex{},
		markup:     NewMarkup(),
	}
}

func (s *StatusBar) stopTimer() {
	s.timerMutex.Lock()
	defer s.timerMutex.Unlock()
	if t := s.clearTimer; t != nil {
		t.Stop()
		s.clearTimer = nil
	}
}

func (s *StatusBar) setClearTimer(t *time.Timer) {
	s.timerMutex.Lock()
	defer s.timerMutex.Unlock()
	s.clearTimer = t
}

// StatusMessage sets a new status message for the given duration
func (s *StatusBar) StatusMessage(msg string, clearDelay time.Duration) {
	s.stopTimer()
	s.lastMessage = msg
	// if everything is successful AND the clearDelay timer is specified,
	// then set a timer to clear the status
	if clearDelay != 0 {
		s.setClearTimer(time.AfterFunc(clearDelay, func() {
			clearMessage := strings.Repeat(" ", len(s.lastMessage))
			s.lastMessage = ""
			renderString(0, s.screenPos, string(clearMessage), termbox.ColorDefault, termbox.ColorDefault)
			termbox.Flush()
		}))
	}
}

//Render renders the status message
func (s *StatusBar) Render() {
	if s.lastMessage != "" {
		w, _ := termbox.Size()
		renderLineWithMarkup(0, s.screenPos, w, s.lastMessage, s.markup)
	}
}
