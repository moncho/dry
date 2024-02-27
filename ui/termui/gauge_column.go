package termui

import (
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

// GaugeColumn is a termui.Gauge to be used as a Grid column. It is
// borderless, has a height of 1 and its label is left-aligned.
type GaugeColumn struct {
	termui.Gauge
}

// NewThemedGaugeColumn creates a new GaugeColumn using the given theme
func NewThemedGaugeColumn(theme *ui.ColorTheme) *GaugeColumn {
	c := NewGaugeColumn()
	c.Bg = termui.Attribute(theme.Bg)
	return c
}

// NewGaugeColumn creates a new GaugeColumn
func NewGaugeColumn() *GaugeColumn {
	g := termui.NewGauge()
	g.Height = 1
	g.Border = false
	g.Percent = 0
	g.PaddingBottom = 0

	return &GaugeColumn{*g}
}

// Reset resets this GaugeColumn
func (w *GaugeColumn) Reset() {
	w.Percent = 0
}
