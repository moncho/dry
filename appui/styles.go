package appui

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles derived from the active theme.
var (
	HeaderStyle      lipgloss.Style
	FooterStyle      lipgloss.Style
	SelectedRowStyle lipgloss.Style
	TableHeaderStyle lipgloss.Style
	InfoStyle        lipgloss.Style
)

func init() {
	InitStyles()
}

// InitStyles rebuilds all derived styles from DryTheme.
// Call after rotating the color theme.
func InitStyles() {
	HeaderStyle = lipgloss.NewStyle().
		Foreground(DryTheme.Fg).
		Background(DryTheme.Header)
	FooterStyle = lipgloss.NewStyle().
		Foreground(DryTheme.Fg).
		Background(DryTheme.Footer)
	SelectedRowStyle = lipgloss.NewStyle().
		Background(DryTheme.CursorLineBg).
		Foreground(DryTheme.Fg)
	TableHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(DryTheme.FgSubtle)
	InfoStyle = lipgloss.NewStyle().
		Foreground(DryTheme.Info)
}

// ColorFg applies a foreground color to text using targeted ANSI sequences
// that only reset the foreground (SGR 39), not the full SGR reset. This
// preserves any outer background color (e.g. the selected row highlight).
func ColorFg(text string, c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[39m", r>>8, g>>8, b>>8, text)
}
