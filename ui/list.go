package ui

import "github.com/gizak/termui"

// NewList returns a new list using the given ColorTheme
func NewList(theme *ColorTheme) *termui.List {
	l := termui.NewList()

	l.Bg = termui.Attribute(theme.Bg)
	l.ItemBgColor = termui.Attribute(theme.Bg)
	l.ItemFgColor = termui.Attribute(theme.Fg)
	return l
}
