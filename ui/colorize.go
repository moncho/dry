package ui

import "charm.land/lipgloss/v2"

// Helper styles using CharmTone palette from Crush.
var (
	blueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B50FF"))  // Charple
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#EB4268"))  // Sriracha
	whiteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#DFDBDD"))  // Ash
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E8FE96"))  // Zest
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#68FFD6"))  // Bok
)

// Blue styles the given text blue.
func Blue(text string) string { return blueStyle.Render(text) }

// Red styles the given text red.
func Red(text string) string { return redStyle.Render(text) }

// White styles the given text white.
func White(text string) string { return whiteStyle.Render(text) }

// Yellow styles the given text yellow.
func Yellow(text string) string { return yellowStyle.Render(text) }

// Cyan styles the given text cyan.
func Cyan(text string) string { return cyanStyle.Render(text) }
