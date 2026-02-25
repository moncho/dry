package appui

import (
	"time"

	"charm.land/lipgloss/v2"
)

// MessageBarModel displays timed status messages.
type MessageBarModel struct {
	text   string
	expiry time.Time
	width  int
}

// SetMessage sets a status message that auto-clears after the given duration.
func (m *MessageBarModel) SetMessage(text string, duration time.Duration) {
	m.text = text
	m.expiry = time.Now().Add(duration)
}

// SetWidth updates the message bar width.
func (m *MessageBarModel) SetWidth(w int) {
	m.width = w
}

// View renders the message bar, returning empty string if expired.
func (m MessageBarModel) View() string {
	if m.text == "" || time.Now().After(m.expiry) {
		return ""
	}
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("25")).
		MaxWidth(m.width)
	return style.Render(m.text)
}
