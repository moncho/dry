package appui

import "github.com/moncho/dry/ui/termui"

// defaultMonitorTableHeader is the default header for the container monitor table
var defaultMonitorTableHeader = NewMonitorTableHeader()

// MonitorTableHeader is the header for container monitor tables
type MonitorTableHeader struct {
	*termui.TableHeader
}

// NewMonitorTableHeader creates a table header for the monitor screen
func NewMonitorTableHeader() *MonitorTableHeader {
	fields := []string{"NAME", "CPU", "MEM", "NET RX/TX", "BLOCK I/O"}

	header := termui.NewHeader(DryTheme)
	header.ColumnSpacing = DefaultColumnSpacing
	//Status indicator header
	header.AddFixedWidthColumn("", 2)
	header.AddFixedWidthColumn("CONTAINER", IDColumnWidth)
	for _, f := range fields {
		header.AddColumn(f)
	}
	header.AddFixedWidthColumn("PIDS", 5)
	header.AddFixedWidthColumn("UPTIME", IDColumnWidth)
	return &MonitorTableHeader{header}
}
