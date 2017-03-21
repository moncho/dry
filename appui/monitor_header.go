package appui

import ui "github.com/gizak/termui"

const (
	columnSpacing = 1
)

//DefaultMonitorTableHeader is the default header for the container monitor table
var DefaultMonitorTableHeader ui.GridBufferer = newMonitorTableHeader()

type monitorTableHeader struct {
	x, y          int
	height, width int
	pars          []*ui.Par
}

func newMonitorTableHeader() *monitorTableHeader {
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
	x := ch.x
	ch.width = w
	//Set width on each par
	iw := calcItemWidth(w, len(ch.pars))
	for _, col := range ch.pars {
		col.SetX(x)
		col.SetWidth(iw)
		x += iw + columnSpacing
	}
}

func (ch *monitorTableHeader) SetX(x int) {
	ch.x = x
}

func (ch *monitorTableHeader) SetY(y int) {
	for _, p := range ch.pars {
		p.SetY(y)
	}
	ch.y = y
}

func (ch *monitorTableHeader) Buffer() ui.Buffer {
	buf := ui.NewBuffer()
	for _, p := range ch.pars {
		buf.Merge(p.Buffer())
	}
	return buf
}

func (ch *monitorTableHeader) addPar(s string) {
	p := ui.NewPar(s)
	p.Height = ch.height
	p.Border = false
	p.Bg = ui.Attribute(DryTheme.Bg)
	p.TextBgColor = ui.Attribute(DryTheme.Bg)
	p.TextFgColor = ui.Attribute(ui.ColorWhite)
	ch.pars = append(ch.pars, p)
}

func calcItemWidth(width, items int) int {
	spacing := columnSpacing * items
	return (width - spacing) / items
}
