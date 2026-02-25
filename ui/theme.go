package ui

import "image/color"

// Theme defines the color palette for the application.
type Theme struct {
	// Base
	Fg     color.Color
	Bg     color.Color
	DarkBg color.Color

	// Text hierarchy
	FgMuted  color.Color // secondary text
	FgSubtle color.Color // de-emphasized text

	// Accents
	Primary   color.Color // main accent (selection, focus)
	Secondary color.Color // secondary accent
	Tertiary  color.Color // tertiary accent

	// Semantic
	Info    color.Color
	Success color.Color
	Error   color.Color
	Warning color.Color

	// UI elements
	Key          color.Color // labels, key hints
	Prompt       color.Color
	Border       color.Color // borders, separators
	Header       color.Color
	Footer       color.Color
	CursorLineBg color.Color
}
