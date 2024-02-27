package ui

import "github.com/gizak/termui"

// NewPar creates a new termui paragraph with the given content and
// with a look and feel based on the given theme.
func NewPar(s string, theme *ColorTheme) *termui.Paragraph {
	p := termui.NewParagraph(s)
	p.Bg = termui.Attribute(theme.Bg)
	p.BorderFg = termui.Attribute(theme.Fg)
	p.BorderBg = termui.Attribute(theme.Bg)
	p.TextFgColor = termui.Attribute(theme.Fg)
	p.TextBgColor = termui.Attribute(theme.Bg)
	p.BorderLabelFg = termui.Attribute(theme.Fg)
	p.BorderLabelBg = termui.Attribute(theme.Bg)
	return p
}
