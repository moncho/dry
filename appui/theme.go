package appui

import (
	"charm.land/lipgloss/v2"
	"github.com/moncho/dry/ui"
)

// CharmTone palette — inspired by github.com/charmbracelet/crush.
var (
	pepper   = lipgloss.Color("#201F26")
	charcoal = lipgloss.Color("#3A3943")
	charple  = lipgloss.Color("#D08850")
	dolly    = lipgloss.Color("#E07890")
	bok      = lipgloss.Color("#E8A848")
	malibu   = lipgloss.Color("#00A4FF")
	julep    = lipgloss.Color("#00FFB2")
	sriracha = lipgloss.Color("#EB4268")
	zest     = lipgloss.Color("#E8FE96")
	ash      = lipgloss.Color("#DFDBDD")
	smoke    = lipgloss.Color("#BFBCC8")
	oyster   = lipgloss.Color("#605F6B")
	darkTeal = lipgloss.Color("#182838")
)

// Light palette — mid-tone warm background, near-black text.
var (
	cream     = lipgloss.Color("#C0B8B0")
	inkBrown  = lipgloss.Color("#000000")
	medBrown  = lipgloss.Color("#1A1510")
	ltBrown   = lipgloss.Color("#3A3530")
	lightGrey = lipgloss.Color("#B0A8A0")
	ltBorder  = lipgloss.Color("#A8A098")
	warmTan   = lipgloss.Color("#E0D8C8")
	dkAmber   = lipgloss.Color("#6B3000")
	dkMagenta = lipgloss.Color("#781878")
	dkTeal    = lipgloss.Color("#004830")
	dkBlue    = lipgloss.Color("#003870")
	dkGreen   = lipgloss.Color("#005030")
	dkRed     = lipgloss.Color("#801030")
	dkGold    = lipgloss.Color("#504000")
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
	CursorLineBg: darkTeal,
}

// CrushLight is the warm light theme.
var CrushLight = &ui.Theme{
	Fg:     inkBrown,
	Bg:     cream,
	DarkBg: cream,

	FgMuted:  medBrown,
	FgSubtle: ltBrown,

	Primary:   dkAmber,
	Secondary: dkMagenta,
	Tertiary:  dkTeal,

	Info:    dkBlue,
	Success: dkGreen,
	Error:   dkRed,
	Warning: dkGold,

	Key:          dkTeal,
	Prompt:       dkMagenta,
	Border:       ltBorder,
	Header:       lightGrey,
	Footer:       lightGrey,
	CursorLineBg: warmTan,
}

// DryTheme is the active theme for dry.
var DryTheme = CrushDark

// ColorThemes holds the list of dry color themes.
var ColorThemes = []*ui.Theme{CrushLight, CrushDark}

// RotateColorTheme changes the color theme to the next one in the
// rotation order.
func RotateColorTheme() {
	if DryTheme == ColorThemes[0] {
		DryTheme = ColorThemes[1]
	} else {
		DryTheme = ColorThemes[0]
	}
}
