package appui

import (
	"time"
)

// MessageBarModel holds timed status messages displayed in the header separator.
type MessageBarModel struct {
	text   string
	expiry time.Time
}

// SetMessage sets a status message that auto-clears after the given duration.
func (m *MessageBarModel) SetMessage(text string, duration time.Duration) {
	m.text = text
	m.expiry = time.Now().Add(duration)
}

// Message returns the active message text, or "" if expired/unset.
func (m MessageBarModel) Message() string {
	if m.text == "" || time.Now().After(m.expiry) {
		return ""
	}
	return m.text
}
