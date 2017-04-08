package appui

import (
	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

const (
	columnSpacing        = 1
	containerColumnWidth = 12
)

//defaultMonitorTableHeader is the default header for the container monitor table
var defaultMonitorTableHeader = NewMonitorTableHeader()

//MonitorTableHeader is the header for container monitor tables
type MonitorTableHeader struct {
	X, Y          int
	height, width int
	pars          []*termui.Par
}

//NewMonitorTableHeader creates a table header for the monitor screen
func NewMonitorTableHeader() *MonitorTableHeader {
	fields := []string{"CONTAINER", "NAME", "CPU", "MEM", "NET RX/TX", "BLOCK I/O", "PIDS", "UPTIME"}
	ch := &MonitorTableHeader{}
	ch.height = 1
	for _, f := range fields {
		ch.addPar(f)
	}
	return ch
}

//GetHeight return this header's height
func (ch *MonitorTableHeader) GetHeight() int {
	return ch.height
}

//SetWidth set the width of this header
func (ch *MonitorTableHeader) SetWidth(w int) {
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

//SetX sets the X position of this header
func (ch *MonitorTableHeader) SetX(x int) {
	ch.X = x
}

//SetY sets the Y position of this header
func (ch *MonitorTableHeader) SetY(y int) {
	for _, p := range ch.pars {
		p.SetY(y)
	}
	ch.Y = y
}

//Buffer returns the content of this header as a buffer
func (ch *MonitorTableHeader) Buffer() termui.Buffer {
	buf := termui.NewBuffer()
	for _, p := range ch.pars {
		buf.Merge(p.Buffer())
	}
	return buf
}

func (ch *MonitorTableHeader) addPar(s string) {
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
