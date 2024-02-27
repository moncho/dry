package appui

import (
	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

const (
	//Running is the color used to identify a running element (e.g container, task)
	Running = termui.Attribute(ui.Color108)
	//NotRunning is the color used to identify a non-running element
	NotRunning = termui.Attribute(ui.Color161)
)

// Default16 default theme for 16-color mode
var Default16 = &ui.ColorTheme{
	Fg:           ui.ColorWhite,
	Bg:           ui.ColorBlack,
	DarkBg:       ui.ColorBlack,
	Prompt:       ui.ColorBlue,
	Key:          ui.ColorGreen,
	Current:      ui.ColorYellow,
	CurrentMatch: ui.ColorGreen,
	Spinner:      ui.ColorGreen,
	Info:         ui.ColorWhite,
	Cursor:       ui.ColorRed,
	Selected:     ui.ColorPurple,
	Header:       ui.ColorLime,
	Footer:       ui.ColorLime}

// Black256 black bg theme for 256-color mode
var Black256 = &ui.ColorTheme{
	Fg:           ui.Color255,
	Bg:           ui.ColorBlack,
	DarkBg:       ui.ColorBlack,
	Prompt:       ui.Color110,
	Key:          ui.Color108,
	Current:      ui.Color254,
	CurrentMatch: ui.Color151,
	Spinner:      ui.Color148,
	Info:         ui.Color144,
	Cursor:       ui.Color161,
	Selected:     ui.Color168,
	Header:       ui.Color25,
	Footer:       ui.Color25}

// Dark256 dark theme for 256-color mode
var Dark256 = &ui.ColorTheme{
	Fg:           ui.Color255,
	Bg:           ui.Color234,
	DarkBg:       ui.ColorBlack,
	Prompt:       ui.Color110,
	Key:          ui.Color108,
	Current:      ui.Color254,
	CurrentMatch: ui.Color151,
	Spinner:      ui.Color148,
	Info:         ui.Color144,
	Cursor:       ui.Color161,
	Selected:     ui.Color168,
	Header:       ui.Color25,
	Footer:       ui.Color25,
	ListItem:     ui.Color181,
	CursorLineBg: ui.Color25}

// Light256 light theme for 256-color mode
var Light256 = &ui.ColorTheme{
	Fg:           ui.Color241,
	Bg:           ui.Color231,
	DarkBg:       ui.Color251,
	Prompt:       ui.Color25,
	Key:          ui.Color66,
	Current:      ui.Color237,
	CurrentMatch: ui.Color23,
	Spinner:      ui.Color65,
	Info:         ui.Color101,
	Cursor:       ui.Color161,
	Selected:     ui.Color168,
	Header:       ui.Color31,
	Footer:       ui.Color31}

// DryTheme is the active theme for dry
var DryTheme = Dark256

// ColorThemes holds the list of dry color themes
var ColorThemes = []*ui.ColorTheme{Black256, Dark256}

// RotateColorTheme changes the color theme to the next one in the
// rotation order.
func RotateColorTheme() {
	if DryTheme == ColorThemes[0] {
		DryTheme = ColorThemes[1]
	} else {
		DryTheme = ColorThemes[0]
	}
}
