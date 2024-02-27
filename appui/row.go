package appui

import (
	"image"

	termui "github.com/gizak/termui"
	drytermui "github.com/moncho/dry/ui/termui"
)

// Row is a base row implementation
type Row struct {
	ParColumns []*drytermui.ParColumn
	drytermui.Row
}

// Highlighted marks this rows as being highlighted
func (row *Row) Highlighted() {
	row.changeTextColor(
		termui.Attribute(DryTheme.Fg),
		termui.Attribute(DryTheme.CursorLineBg))
}

// NotHighlighted marks this rows as being not highlighted
func (row *Row) NotHighlighted() {

	row.changeTextColor(
		termui.Attribute(DryTheme.ListItem),
		termui.Attribute(DryTheme.Bg))
}

// Buffer returns this Row data as a termui.Buffer
func (row *Row) Buffer() termui.Buffer {
	buf := termui.NewBuffer()
	//This set the background of the whole row
	buf.Area.Min = image.Point{row.X, row.Y}
	buf.Area.Max = image.Point{row.X + row.Width, row.Y + row.Height}
	buf.Fill(' ', row.ParColumns[0].TextFgColor, row.ParColumns[0].TextBgColor)

	for _, col := range row.Columns {
		buf.Merge(col.Buffer())
	}
	return buf
}

func (row *Row) changeTextColor(fg, bg termui.Attribute) {
	for _, c := range row.ParColumns {
		c.TextFgColor = fg
		c.TextBgColor = bg
	}
}
