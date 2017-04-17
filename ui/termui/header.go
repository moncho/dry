package termui

import (
	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

//TableHeader is a table header widget
type TableHeader struct {
	X, Y          int
	Height, Width int
	Columns       []*termui.Par
	ColumnSpacing int
	ColumnWidth   int
	Theme         *ui.ColorTheme
}

//NewHeader creates a header of height 1 that uses the given Theme
func NewHeader(Theme *ui.ColorTheme) *TableHeader {
	return &TableHeader{Height: 1, Theme: Theme}
}

//GetHeight return this header's height
func (th *TableHeader) GetHeight() int {
	return th.Height
}

//SetWidth set the width of this header
func (th *TableHeader) SetWidth(w int) {
	x := th.X
	th.Width = w
	//Set width on each par
	iw := th.CalcColumnWidth(th.ColumnCount())
	for _, col := range th.Columns {
		col.SetX(x)
		col.SetWidth(iw)
		x += iw + th.ColumnSpacing
	}
}

//SetX sets the X position of this header
func (th *TableHeader) SetX(x int) {
	th.X = x
}

//SetY sets the Y position of this header
func (th *TableHeader) SetY(y int) {
	for _, p := range th.Columns {
		p.SetY(y)
	}
	th.Y = y
}

//Buffer returns the content of this header as a buffer
func (th *TableHeader) Buffer() termui.Buffer {
	buf := termui.NewBuffer()
	for _, p := range th.Columns {
		buf.Merge(p.Buffer())
	}
	return buf
}

//AddColumn adds a column to this header
func (th *TableHeader) AddColumn(s string) {
	p := termui.NewPar(s)
	p.Height = th.Height
	p.Border = false
	p.Bg = termui.Attribute(th.Theme.Bg)
	p.TextBgColor = termui.Attribute(th.Theme.Bg)
	p.TextFgColor = termui.ColorWhite
	th.Columns = append(th.Columns, p)
}

//CalcColumnWidth calculates column width for this header
func (th *TableHeader) CalcColumnWidth(colCount int) int {
	spacing := th.ColumnSpacing * colCount
	return (th.Width - spacing) / colCount
}

//ColumnCount returns the number of columns on this header
func (th *TableHeader) ColumnCount() int {
	return len(th.Columns)
}
