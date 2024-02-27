package termui

import (
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

// ParColumn is a termui.Par that can be used in a grid to show text
type ParColumn struct {
	termui.Paragraph
}

// NewThemedParColumn creates a new paragraph column with the given text using the given color theme
func NewThemedParColumn(theme *ui.ColorTheme, s string) *ParColumn {
	p := NewParColumn(s)
	p.Bg = termui.Attribute(theme.Bg)
	p.TextBgColor = termui.Attribute(theme.Bg)
	p.TextFgColor = termui.Attribute(theme.ListItem)
	return p
}

// NewParColumn creates a new paragraph column with the given text
func NewParColumn(s string) *ParColumn {
	p := termui.NewParagraph(s)
	p.Border = false

	return &ParColumn{*p}
}

// Reset resets the text on this Par
func (w *ParColumn) Reset() {
	w.Content("-")
}

// Content sets the text of this Par to the given content
func (w *ParColumn) Content(s string) {
	w.Text = s
	w.Width = len(w.Text)
}

// SetWidth sets par width.
func (w *ParColumn) SetWidth(width int) {
	contentWidth := len(w.Text)
	if width > contentWidth {
		w.Width = contentWidth
	} else {
		w.Width = width
	}
}
