package appui

import "github.com/moncho/dry/ui/termui"

//defaultMonitorTableHeader is the default header for the container monitor table
var defaultMonitorTableHeader = NewMonitorTableHeader()

//MonitorTableHeader is the header for container monitor tables
type MonitorTableHeader struct {
	*termui.TableHeader
}

//NewMonitorTableHeader creates a table header for the monitor screen
func NewMonitorTableHeader() *MonitorTableHeader {
	fields := []string{"CONTAINER", "NAME", "CPU", "MEM", "NET RX/TX", "BLOCK I/O", "PIDS", "UPTIME"}

	header := termui.NewHeader(DryTheme)
	header.ColumnSpacing = DefaultColumnSpacing
	for _, f := range fields {
		header.AddColumn(f)
	}
	return &MonitorTableHeader{header}
}

//SetWidth set the width of this header
func (ch *MonitorTableHeader) SetWidth(w int) {
	x := ch.X
	ch.Width = w
	//Set width on each par
	iw := ch.CalcColumnWidth(ch.ColumnCount() - 1)
	for _, col := range ch.Columns {
		col.SetX(x)
		if col.Text != "CONTAINER" {
			col.SetWidth(iw)
			x += iw + ch.ColumnSpacing

		} else {
			col.SetWidth(IDColumnWidth)
			x += IDColumnWidth + ch.ColumnSpacing
		}
	}
}
