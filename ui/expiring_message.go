package ui

import (
	"sync"
	"time"
)

// ExpiringMessageWidget shows some text for an amount time then clears itself
type ExpiringMessageWidget struct {
	y           int
	screenWidth int
	clearTimer  *time.Timer
	screen      *Screen

	sync.RWMutex
	message string
}

// NewExpiringMessageWidget creates a new ExpiringMessageWidget struct
func NewExpiringMessageWidget(y int, screen *Screen) *ExpiringMessageWidget {
	return &ExpiringMessageWidget{
		y:          y,
		screen:     screen,
		clearTimer: nil,
	}

}

// Pause pauses this widget from showing any output, setting a new status
// message will activate it again
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
		ActiveScreen.Fill(0, s.y, len(msg), 1, ' ')
	})

}

// Render renders the status message
func (s *ExpiringMessageWidget) Render() {
	s.RLock()
	s.RUnlock()
	if s.message == "" {
		return
	}
	s.screen.RenderLine(0, s.y, s.message)
}
