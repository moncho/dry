package ui

import "image/color"

// Theme defines the color palette for the application.
type Theme struct {
	Fg           color.Color
	Bg           color.Color
	DarkBg       color.Color
	Prompt       color.Color
	Key          color.Color
	Current      color.Color
	CurrentMatch color.Color
	Spinner      color.Color
	Info         color.Color
	Cursor       color.Color
	Selected     color.Color
	Header       color.Color
	Footer       color.Color
	ListItem     color.Color
	CursorLineBg color.Color
}
