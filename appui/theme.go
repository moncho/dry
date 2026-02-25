package appui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/moncho/dry/ui"
)

// Running is the color used to identify a running element (e.g container, task)
var Running color.Color = lipgloss.Color("108")

// NotRunning is the color used to identify a non-running element
var NotRunning color.Color = lipgloss.Color("161")

// Black256 black bg theme for 256-color mode
var Black256 = &ui.Theme{
	Fg:           lipgloss.Color("255"),
	Bg:           lipgloss.Color("0"),
	DarkBg:       lipgloss.Color("0"),
	Prompt:       lipgloss.Color("110"),
	Key:          lipgloss.Color("108"),
	Current:      lipgloss.Color("254"),
	CurrentMatch: lipgloss.Color("151"),
	Spinner:      lipgloss.Color("148"),
	Info:         lipgloss.Color("144"),
	Cursor:       lipgloss.Color("161"),
	Selected:     lipgloss.Color("168"),
	Header:       lipgloss.Color("25"),
	Footer:       lipgloss.Color("25"),
}

// Dark256 dark theme for 256-color mode
var Dark256 = &ui.Theme{
	Fg:           lipgloss.Color("255"),
	Bg:           lipgloss.Color("234"),
	DarkBg:       lipgloss.Color("0"),
	Prompt:       lipgloss.Color("110"),
	Key:          lipgloss.Color("108"),
	Current:      lipgloss.Color("254"),
	CurrentMatch: lipgloss.Color("151"),
	Spinner:      lipgloss.Color("148"),
	Info:         lipgloss.Color("144"),
	Cursor:       lipgloss.Color("161"),
	Selected:     lipgloss.Color("168"),
	Header:       lipgloss.Color("25"),
	Footer:       lipgloss.Color("25"),
	CursorLineBg: lipgloss.Color("25"),
}

// DryTheme is the active theme for dry
var DryTheme = Dark256

// ColorThemes holds the list of dry color themes
var ColorThemes = []*ui.Theme{Black256, Dark256}

// RotateColorTheme changes the color theme to the next one in the
// rotation order.
func RotateColorTheme() {
	if DryTheme == ColorThemes[0] {
		DryTheme = ColorThemes[1]
	} else {
		DryTheme = ColorThemes[0]
	}
}
