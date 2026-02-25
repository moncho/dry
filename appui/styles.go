package appui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

// WidgetHeaderOpts configures the widget header bar.
type WidgetHeaderOpts struct {
	Icon     string      // e.g. "🐳"
	Title    string      // e.g. "Containers"
	Total    int         // total row count
	Filtered int         // visible (filtered) row count; same as Total when no filter
	Filter   string      // active filter text, empty if none
	Width    int         // full terminal width for padding
	Accent   color.Color // icon/filter accent color
}

// RenderWidgetHeader renders a full-width accent bar with icon, title, and count.
func RenderWidgetHeader(o WidgetHeaderOpts) string {
	bg := lipgloss.NewStyle().Background(DryTheme.Header)
	iconStyle := lipgloss.NewStyle().
		Foreground(o.Accent).
		Background(DryTheme.Header)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(DryTheme.Fg).
		Background(DryTheme.Header)
	countStyle := lipgloss.NewStyle().
		Foreground(DryTheme.FgMuted).
		Background(DryTheme.Header)
	filterStyle := lipgloss.NewStyle().
		Foreground(o.Accent).
		Background(DryTheme.Header)
	sepStyle := lipgloss.NewStyle().
		Foreground(DryTheme.FgSubtle).
		Background(DryTheme.Header)

	var b strings.Builder
	b.WriteString(bg.Render(" "))
	b.WriteString(iconStyle.Render(o.Icon))
	b.WriteString(bg.Render(" "))
	b.WriteString(titleStyle.Render(o.Title))
	b.WriteString(bg.Render("  "))

	if o.Total != o.Filtered {
		b.WriteString(countStyle.Render(fmt.Sprintf("%d", o.Total)))
		b.WriteString(sepStyle.Render(" › "))
		b.WriteString(countStyle.Render(fmt.Sprintf("%d", o.Filtered)))
	} else {
		b.WriteString(countStyle.Render(fmt.Sprintf("%d", o.Total)))
	}

	if o.Filter != "" {
		b.WriteString(bg.Render("  "))
		b.WriteString(sepStyle.Render("│"))
		b.WriteString(bg.Render(" "))
		b.WriteString(sepStyle.Render("filter: "))
		b.WriteString(filterStyle.Render(o.Filter))
	}

	line := b.String()
	w := ansi.StringWidth(line)
	if w > o.Width {
		line = ansi.Truncate(line, o.Width, "")
	} else if w < o.Width {
		line += bg.Render(strings.Repeat(" ", o.Width-w))
	}
	return line + "\n"
}

// ColorFg applies a foreground color to text using targeted ANSI sequences
// that only reset the foreground (SGR 39), not the full SGR reset. This
// preserves any outer background color (e.g. the selected row highlight).
func ColorFg(text string, c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[39m", r>>8, g>>8, b>>8, text)
}
