package appui

import (
	"charm.land/lipgloss/v2"
	"github.com/moncho/dry/ui"
)

// CharmTone palette — inspired by github.com/charmbracelet/crush.
// Unexported: components should use DryTheme fields instead.
var (
	pepper   = lipgloss.Color("#201F26")
	charcoal = lipgloss.Color("#3A3943")
	charple  = lipgloss.Color("#6B50FF")
	dolly    = lipgloss.Color("#FF60FF")
	bok      = lipgloss.Color("#68FFD6")
	malibu   = lipgloss.Color("#00A4FF")
	julep    = lipgloss.Color("#00FFB2")
	sriracha = lipgloss.Color("#EB4268")
	zest     = lipgloss.Color("#E8FE96")
	ash      = lipgloss.Color("#DFDBDD")
	smoke    = lipgloss.Color("#BFBCC8")
	squid    = lipgloss.Color("#858392")
	oyster   = lipgloss.Color("#605F6B")
)

// CrushDark is the Crush-inspired dark theme.
var CrushDark = &ui.Theme{
	Fg:     ash,
	Bg:     pepper,
	DarkBg: pepper,

	FgMuted:  smoke,
	FgSubtle: oyster,

	Primary:   charple,
	Secondary: dolly,
	Tertiary:  bok,

	Info:    malibu,
	Success: julep,
	Error:   sriracha,
	Warning: zest,

	Key:          bok,
	Prompt:       dolly,
	Border:       charcoal,
	Header:       charcoal,
	Footer:       charcoal,
	CursorLineBg: charple,
}

// CrushBlack is a variant with pure-black background.
var CrushBlack = &ui.Theme{
	Fg:     ash,
	Bg:     lipgloss.Color("#000000"),
	DarkBg: lipgloss.Color("#000000"),

	FgMuted:  smoke,
	FgSubtle: oyster,

	Primary:   charple,
	Secondary: dolly,
	Tertiary:  bok,

	Info:    malibu,
	Success: julep,
	Error:   sriracha,
	Warning: zest,

	Key:          bok,
	Prompt:       dolly,
	Border:       charcoal,
	Header:       charcoal,
	Footer:       charcoal,
	CursorLineBg: charple,
}

// DryTheme is the active theme for dry.
var DryTheme = CrushDark

// ColorThemes holds the list of dry color themes.
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
