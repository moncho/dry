package appui

import (
	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

const (
	columnSpacing        = 1
	containerColumnWidth = 12
)

//DefaultMonitorTableHeader is the default header for the container monitor table
var DefaultMonitorTableHeader = NewMonitorTableHeader()

type monitorTableHeader struct {
	X, Y          int
	height, width int
	pars          []*termui.Par
}

func NewMonitorTableHeader() *monitorTableHeader {
	fields := []string{"CONTAINER", "NAME", "CPU", "MEM", "NET RX/TX", "BLOCK I/O", "PIDS"}
	ch := &monitorTableHeader{}
	ch.height = 1
	for _, f := range fields {
		ch.addPar(f)
	}
	return ch
}

func (ch *monitorTableHeader) GetHeight() int {
	return ch.height
}

func (ch *monitorTableHeader) SetWidth(w int) {
	x := ch.X
	ch.width = w
	//Set width on each par
	iw := calcItemWidth(w, len(ch.pars)-1)
	for _, col := range ch.pars {
		col.SetX(x)
		if col.Text != "CONTAINER" {
			col.SetWidth(iw)
			x += iw + columnSpacing

		} else {
			col.SetWidth(containerColumnWidth)
			x += containerColumnWidth + columnSpacing
		}

	}
}

func (ch *monitorTableHeader) SetX(x int) {
	ch.X = x
}

func (ch *monitorTableHeader) SetY(y int) {
	for _, p := range ch.pars {
		p.SetY(y)
	}
	ch.Y = y
}

func (ch *monitorTableHeader) Buffer() termui.Buffer {
	buf := termui.NewBuffer()
	for _, p := range ch.pars {
		buf.Merge(p.Buffer())
	}
	return buf
}

func (ch *monitorTableHeader) addPar(s string) {
	p := termui.NewPar(s)
	p.Height = ch.height
	p.Border = false
	p.Bg = termui.Attribute(DryTheme.Bg)
	p.TextBgColor = termui.Attribute(DryTheme.Bg)
	p.TextFgColor = termui.Attribute(ui.ColorWhite)
	ch.pars = append(ch.pars, p)
}

func calcItemWidth(width, items int) int {
	spacing := columnSpacing * items
	return (width - spacing) / items
}
