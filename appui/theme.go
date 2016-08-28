package appui

import (
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//Default16 default theme for 16-color mode
var Default16 = &ui.ColorTheme{
	Fg:           15,
	Bg:           0,
	DarkBg:       ui.Color(termbox.ColorBlack),
	Prompt:       ui.Color(termbox.ColorBlue),
	Key:          ui.Color(termbox.ColorGreen),
	Current:      ui.Color(termbox.ColorYellow),
	CurrentMatch: ui.Color(termbox.ColorGreen),
	Spinner:      ui.Color(termbox.ColorGreen),
	Info:         ui.Color(termbox.ColorWhite),
	Cursor:       ui.Color(termbox.ColorRed),
	Selected:     ui.Color(termbox.ColorMagenta),
	Header:       ui.Color(termbox.ColorCyan),
	Footer:       ui.Color(termbox.ColorCyan)}

//Dark256 dark theme for 256-color mode
var Dark256 = &ui.ColorTheme{
	Fg:           15,
	Bg:           0,
	DarkBg:       236,
	Prompt:       110,
	Key:          108,
	Current:      254,
	CurrentMatch: 151,
	Spinner:      148,
	Info:         144,
	Cursor:       161,
	Selected:     168,
	Header:       ui.MenuBarBackgroundColor,
	Footer:       ui.MenuBarBackgroundColor}

//Light256 light theme for 256-color mode
var Light256 = &ui.ColorTheme{
	Fg:           15,
	Bg:           0,
	DarkBg:       251,
	Prompt:       25,
	Key:          66,
	Current:      237,
	CurrentMatch: 23,
	Spinner:      65,
	Info:         101,
	Cursor:       161,
	Selected:     168,
	Header:       31,
	Footer:       31}

//DryTheme is the active theme for dry
var DryTheme = Dark256
