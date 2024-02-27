package appui

import (
	"fmt"
	"image"
	"strconv"
	"sync"
	"time"

	units "github.com/docker/go-units"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
	"github.com/moncho/dry/ui"
	drytermui "github.com/moncho/dry/ui/termui"
)

const inactiveRowColor = termui.Attribute(ui.Color244)
const inactiveRowText = "-"

// ContainerStatsRow is a Grid row showing runtime information about a container
type ContainerStatsRow struct {
	container *docker.Container
	Status    *drytermui.ParColumn
	Name      *drytermui.ParColumn
	ID        *drytermui.ParColumn
	CPU       *drytermui.GaugeColumn
	Memory    *drytermui.GaugeColumn
	Net       *drytermui.ParColumn
	Block     *drytermui.ParColumn
	Pids      *drytermui.ParColumn
	Uptime    *drytermui.ParColumn
	PidsVal   uint64
	UptimeVal time.Time

	drytermui.Row
	sync.RWMutex
}

// NewContainerStatsRow creats a new ContainerStatsRow widget
func NewContainerStatsRow(container *docker.Container, table drytermui.Table) *ContainerStatsRow {
	cf := formatter.NewContainerFormatter(container, true)
	row := ContainerStatsRow{
		container: container,
		Status:    drytermui.NewThemedParColumn(DryTheme, statusSymbol),
		Name:      drytermui.NewThemedParColumn(DryTheme, cf.Names()),
		ID:        drytermui.NewThemedParColumn(DryTheme, cf.ID()),
		CPU:       drytermui.NewThemedGaugeColumn(DryTheme),
		Memory:    drytermui.NewThemedGaugeColumn(DryTheme),
		Net:       drytermui.NewThemedParColumn(DryTheme, inactiveRowText),
		Block:     drytermui.NewThemedParColumn(DryTheme, inactiveRowText),
		Pids:      drytermui.NewThemedParColumn(DryTheme, inactiveRowText),
		Uptime:    drytermui.NewThemedParColumn(DryTheme, container.Status),
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.Status,
		row.ID,
		row.Name,
		row.CPU,
		row.Memory,
		row.Net,
		row.Block,
		row.Pids,
		row.Uptime,
	}
	if !docker.IsContainerRunning(container) {
		row.markAsNotRunning()
	} else {
		row.Status.TextFgColor = Running
	}
	return &row

}

// Highlighted marks this rows as being highlighted
func (row *ContainerStatsRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(DryTheme.Fg),
		termui.Attribute(DryTheme.CursorLineBg))
}

// NotHighlighted marks this rows as being not highlighted
func (row *ContainerStatsRow) NotHighlighted() {
	row.changeTextColor(
		termui.Attribute(DryTheme.ListItem),
		termui.Attribute(DryTheme.Bg))
}

// Buffer returns this Row data as a termui.Buffer
func (row *ContainerStatsRow) Buffer() termui.Buffer {
	row.RLock()
	defer row.RUnlock()
	buf := termui.NewBuffer()
	//This set the background of the whole row
	buf.Area.Min = image.Point{row.X, row.Y}
	buf.Area.Max = image.Point{row.X + row.Width, row.Y + row.Height}
	buf.Fill(' ', row.ID.TextFgColor, row.ID.TextBgColor)

	for _, col := range row.Columns {
		buf.Merge(col.Buffer())
	}
	return buf
}

func (row *ContainerStatsRow) changeTextColor(fg, bg termui.Attribute) {
	row.ID.TextFgColor = fg
	row.ID.TextBgColor = bg
	row.Name.TextFgColor = fg
	row.Name.TextBgColor = bg
	row.Net.TextFgColor = fg
	row.Net.TextBgColor = bg
	row.Block.TextFgColor = fg
	row.Block.TextBgColor = bg
	row.Pids.TextFgColor = fg
	row.Pids.TextBgColor = bg

	row.Uptime.TextFgColor = fg
	row.Uptime.TextBgColor = bg
}

// Reset resets row content
func (row *ContainerStatsRow) Reset() {
	row.CPU.Reset()
	row.Memory.Reset()
	row.Net.Reset()
	row.Pids.Reset()
	row.Block.Reset()
	row.Uptime.Reset()
}

// Update updates the content of this row with the given stats
func (row *ContainerStatsRow) Update(stat *docker.Stats) {
	if stat == nil {
		return
	}
	row.Lock()
	defer row.Unlock()
	row.setNet(stat.NetworkRx, stat.NetworkTx)
	row.setCPU(stat.CPUPercentage)
	row.setMem(stat.Memory, stat.MemoryLimit, stat.MemoryPercentage)
	row.setBlockIO(stat.BlockRead, stat.BlockWrite)
	row.setPids(stat.PidsCurrent)
	row.setUptime(row.container.ContainerJSON.State.StartedAt)
}

func (row *ContainerStatsRow) setNet(rx float64, tx float64) {
	row.Net.Content(fmt.Sprintf("%s / %s", units.BytesSize(rx), units.BytesSize(tx)))
}

func (row *ContainerStatsRow) setBlockIO(read float64, write float64) {
	row.Block.Content(fmt.Sprintf("%s / %s", units.BytesSize(read), units.BytesSize(write)))
}
func (row *ContainerStatsRow) setPids(pids uint64) {
	row.PidsVal = pids
	row.Pids.Content(strconv.Itoa(int(pids)))
}

func (row *ContainerStatsRow) setCPU(val float64) {
	row.CPU.Label = fmt.Sprintf("%.2f%%", val)
	cpu := int(val)
	if val > 0 && val < 5 {
		cpu = 5
	} else if val > 100 {
		cpu = 100
	}
	row.CPU.Percent = cpu
	row.CPU.BarColor = percentileToColor(cpu)
}

func (row *ContainerStatsRow) setMem(val float64, limit float64, percent float64) {
	row.Memory.Label = fmt.Sprintf("%s / %s", units.BytesSize(val), units.BytesSize(limit))
	mem := int(percent)
	if mem < 5 {
		mem = 5
	} else if mem > 100 {
		mem = 100
	}
	row.Memory.Percent = mem
	row.Memory.BarColor = percentileToColor(mem)
}

func (row *ContainerStatsRow) setUptime(startedAt string) {
	if startTime, err := time.Parse(time.RFC3339, startedAt); err == nil {
		row.UptimeVal = startTime
		row.Uptime.Text = units.HumanDuration(time.Now().UTC().Sub(startTime))
	} else {
		row.Uptime.Text = ""
	}
}

// markAsNotRunning
func (row *ContainerStatsRow) markAsNotRunning() {
	row.Status.TextFgColor = NotRunning
	row.Name.TextFgColor = inactiveRowColor
	row.ID.TextFgColor = inactiveRowColor
	row.CPU.PercentColor = inactiveRowColor
	row.CPU.Percent = 0
	row.CPU.Label = inactiveRowText
	row.Memory.PercentColor = inactiveRowColor
	row.Memory.Percent = 0
	row.Memory.Label = inactiveRowText
	row.Net.TextFgColor = inactiveRowColor
	row.Net.Text = inactiveRowText
	row.Block.TextFgColor = inactiveRowColor
	row.Block.Text = inactiveRowText
	row.Pids.Text = "0"
	row.Pids.TextFgColor = inactiveRowColor
	row.Uptime.Text = inactiveRowText
	row.Pids.TextFgColor = inactiveRowColor

}

func percentileToColor(n int) termui.Attribute {
	c := ui.Color23
	if n > 90 {
		c = ui.Color161
	} else if n > 60 {
		c = ui.Color131
	}
	return termui.Attribute(c)
}
