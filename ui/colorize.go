package ui

import "charm.land/lipgloss/v2"

var (
	blueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("188"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	whiteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
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
