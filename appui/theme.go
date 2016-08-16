package appui

import (
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//ColorTheme is the color theme for dry
type ColorTheme struct {
	Fg           ui.Color
	Bg           ui.Color
	DarkBg       ui.Color
	Prompt       ui.Color
	Key          ui.Color
	Current      ui.Color
	CurrentMatch ui.Color
	Spinner      ui.Color
	Info         ui.Color
	Cursor       ui.Color
	Selected     ui.Color
	Header       ui.Color
}

//Default16 default theme for 16-color mode
var Default16 = &ColorTheme{
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
	Header:       ui.Color(termbox.ColorCyan)}

//Dark256 dark theme for 256-color mode
var Dark256 = &ColorTheme{
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
	Header:       109}

//Light256 light theme for 256-color mode
var Light256 = &ColorTheme{
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
	Header:       31}

//DryTheme is the active theme for dry
var DryTheme = Dark256
