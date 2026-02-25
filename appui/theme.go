package appui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/moncho/dry/ui"
)

// CharmTone palette — inspired by github.com/charmbracelet/crush
var (
	// Backgrounds
	Pepper   = lipgloss.Color("#201F26") // base bg
	Charcoal = lipgloss.Color("#3A3943") // subtle borders, separators
	Overlay  = lipgloss.Color("#4D4C57") // overlay bg

	// Primary accent
	Charple = lipgloss.Color("#6B50FF") // purple — selection, focused borders

	// Secondary / tertiary accents
	Dolly = lipgloss.Color("#FF60FF") // magenta-pink
	Bok   = lipgloss.Color("#68FFD6") // cyan-green

	// Semantic colors
	Malibu   = lipgloss.Color("#00A4FF") // info blue
	Julep    = lipgloss.Color("#00FFB2") // success green
	Sriracha = lipgloss.Color("#EB4268") // error red
	Zest     = lipgloss.Color("#E8FE96") // warning yellow

	// Text hierarchy
	Ash    = lipgloss.Color("#DFDBDD") // primary text
	Smoke  = lipgloss.Color("#BFBCC8") // secondary text
	Squid  = lipgloss.Color("#858392") // muted text
	Oyster = lipgloss.Color("#605F6B") // subtle text
)

// Running is the color used to identify a running element (e.g container, task)
var Running color.Color = Julep

// NotRunning is the color used to identify a non-running element
var NotRunning color.Color = Sriracha

// CrushDark is the Crush-inspired dark theme
var CrushDark = &ui.Theme{
	Fg:           Ash,
	Bg:           Pepper,
	DarkBg:       Pepper,
	Prompt:       Dolly,
	Key:          Bok,
	Current:      Ash,
	CurrentMatch: Dolly,
	Spinner:      Charple,
	Info:         Malibu,
	Cursor:       Sriracha,
	Selected:     Dolly,
	Header:       Charcoal,
	Footer:       Charcoal,
	CursorLineBg: Charple,
}

// CrushBlack is a variant with pure-black background
var CrushBlack = &ui.Theme{
	Fg:           Ash,
	Bg:           lipgloss.Color("#000000"),
	DarkBg:       lipgloss.Color("#000000"),
	Prompt:       Dolly,
	Key:          Bok,
	Current:      Ash,
	CurrentMatch: Dolly,
	Spinner:      Charple,
	Info:         Malibu,
	Cursor:       Sriracha,
	Selected:     Dolly,
	Header:       Charcoal,
	Footer:       Charcoal,
	CursorLineBg: Charple,
}

// DryTheme is the active theme for dry
var DryTheme = CrushDark

// ColorThemes holds the list of dry color themes
var ColorThemes = []*ui.Theme{CrushBlack, CrushDark}

// RotateColorTheme changes the color theme to the next one in the
// rotation order.
func RotateColorTheme() {
	if DryTheme == ColorThemes[0] {
		DryTheme = ColorThemes[1]
	} else {
		DryTheme = ColorThemes[0]
	}
}
