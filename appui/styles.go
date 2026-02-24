package appui

import "charm.land/lipgloss/v2"

// Styles derived from the active theme.
var (
	HeaderStyle      lipgloss.Style
	FooterStyle      lipgloss.Style
	SelectedRowStyle lipgloss.Style
	TableHeaderStyle lipgloss.Style
	RunningStyle     lipgloss.Style
	StoppedStyle     lipgloss.Style
	InfoStyle        lipgloss.Style
)

func init() {
	InitStyles()
}

// InitStyles rebuilds all derived styles from DryTheme.
// Call after rotating the color theme.
func InitStyles() {
	HeaderStyle = lipgloss.NewStyle().
		Foreground(DryTheme.Fg).
		Background(DryTheme.Header)
	FooterStyle = lipgloss.NewStyle().
		Foreground(DryTheme.Fg).
		Background(DryTheme.Footer)
	SelectedRowStyle = lipgloss.NewStyle().
		Background(DryTheme.CursorLineBg).
		Foreground(DryTheme.Fg)
	TableHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(DryTheme.Header)
	RunningStyle = lipgloss.NewStyle().
		Foreground(Running)
	StoppedStyle = lipgloss.NewStyle().
		Foreground(NotRunning)
	InfoStyle = lipgloss.NewStyle().
		Foreground(DryTheme.Info)
}
