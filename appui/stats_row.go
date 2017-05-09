package appui

import (
	"fmt"
	"strconv"
	"time"

	units "github.com/docker/go-units"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
	"github.com/moncho/dry/ui"
	drytermui "github.com/moncho/dry/ui/termui"
)

var inactiveRowColor = termui.Attribute(ui.Color244)

//ContainerStatsRow is a Grid row showing runtime information about a container
type ContainerStatsRow struct {
	table     drytermui.Table
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

	drytermui.Row
}

//NewContainerStatsRow creats a new ContainerStatsRow widget
func NewContainerStatsRow(container *docker.Container, table drytermui.Table) *ContainerStatsRow {
	cf := formatter.NewContainerFormatter(container, true)
	row := &ContainerStatsRow{
		container: container,
		Status:    drytermui.NewThemedParColumn(DryTheme, statusSymbol),
		Name:      drytermui.NewThemedParColumn(DryTheme, cf.Names()),
		ID:        drytermui.NewThemedParColumn(DryTheme, cf.ID()),
		CPU:       drytermui.NewThemedGaugeColumn(DryTheme),
		Memory:    drytermui.NewThemedGaugeColumn(DryTheme),
		Net:       drytermui.NewThemedParColumn(DryTheme, "-"),
		Block:     drytermui.NewThemedParColumn(DryTheme, "-"),
		Pids:      drytermui.NewThemedParColumn(DryTheme, "-"),
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
	return row

}

//NewSelfUpdatedContainerStatsRow creates a ContainerStatsRow that updates
//itself on stats message sent on the given channel
func NewSelfUpdatedContainerStatsRow(s *docker.StatsChannel, table drytermui.Table) *ContainerStatsRow {
	c := s.Container
	row := NewContainerStatsRow(c, table)

	if docker.IsContainerRunning(c) {
		go func() {
			for stat := range s.Stats {
				row.Update(c, stat)
			}
			row.markAsNotRunning()
		}()
	}
	return row
}

//Highlighted marks this rows as being highlighted
func (row *ContainerStatsRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(DryTheme.Fg),
		termui.Attribute(DryTheme.CursorLineBg))
}

//NotHighlighted marks this rows as being not highlighted
func (row *ContainerStatsRow) NotHighlighted() {
	row.changeTextColor(
		termui.Attribute(DryTheme.ListItem),
		termui.Attribute(DryTheme.Bg))
}

func (row *ContainerStatsRow) changeTextColor(fg, bg termui.Attribute) {
	row.ID.TextFgColor = fg
	row.ID.TextBgColor = bg
}

//Reset resets row content
func (row *ContainerStatsRow) Reset() {
	row.CPU.Reset()
	row.Memory.Reset()
	row.Net.Reset()
	row.Pids.Reset()
	row.Block.Reset()
	row.Uptime.Reset()
}

//Update updates the content of this row with the given stats
func (row *ContainerStatsRow) Update(container *docker.Container, stat *docker.Stats) {
	row.setNet(stat.NetworkRx, stat.NetworkTx)
	row.setCPU(stat.CPUPercentage)
	row.setMem(stat.Memory, stat.MemoryLimit, stat.MemoryPercentage)
	row.setBlockIO(stat.BlockRead, stat.BlockWrite)
	row.setPids(stat.PidsCurrent)
	row.setUptime(container.ContainerJSON.State.StartedAt)
}

func (row *ContainerStatsRow) setNet(rx float64, tx float64) {
	row.Net.Text = fmt.Sprintf("%s / %s", units.BytesSize(rx), units.BytesSize(tx))
}

func (row *ContainerStatsRow) setBlockIO(read float64, write float64) {
	row.Block.Text = fmt.Sprintf("%s / %s", units.BytesSize(read), units.BytesSize(write))
}
func (row *ContainerStatsRow) setPids(pids uint64) {
	row.Pids.Text = strconv.Itoa(int(pids))
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
		row.Uptime.Text = units.HumanDuration(time.Now().UTC().Sub(startTime))
	} else {
		row.Uptime.Text = ""
	}
}

//markAsNotRunning
func (row *ContainerStatsRow) markAsNotRunning() {
	row.Status.TextFgColor = NotRunning
	row.Name.TextFgColor = inactiveRowColor
	row.ID.TextFgColor = inactiveRowColor
	row.CPU.PercentColor = inactiveRowColor
	row.CPU.Percent = 0
	row.CPU.Label = "-"
	row.Memory.PercentColor = inactiveRowColor
	row.Memory.Percent = 0
	row.Memory.Label = "-"
	row.Net.TextFgColor = inactiveRowColor
	row.Net.Text = "-"
	row.Block.TextFgColor = inactiveRowColor
	row.Block.Text = "-"
	row.Pids.Text = "0"
	row.Pids.TextFgColor = inactiveRowColor
	row.Uptime.Text = "-"
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
