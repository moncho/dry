package appui

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/docker/go-units"
	"github.com/gizak/termui"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

type statsRenderer struct {
	stats *drydocker.Stats
}

//NewDockerStatsRenderer creates renderer for docker stats
func NewDockerStatsRenderer(stats *drydocker.Stats) ui.Renderer {
	return &statsRenderer{
		stats: stats,
	}
}

//Render container stats
func (r *statsRenderer) Render() string {
	s := r.stats
	if s == nil {
		return ""
	}
	processList := s.ProcessList

	buf := bytes.NewBufferString("")
	io.WriteString(buf, "<yellow><b>STATS</></>\n")

	w := tabwriter.NewWriter(buf, 22, 0, 1, ' ', 0)
	io.WriteString(w, "<blue>%CPU\tMEM USAGE / LIMIT\t%MEM\tNET I/O\tBLOCK I/O</>\n")
	io.WriteString(
		w,
		fmt.Sprintf("<white>%.2f\t%s / %s\t%.2f\t%s / %s\t%s / %s</>\n\n",
			s.CPUPercentage,
			units.HumanSize(s.Memory), units.HumanSize(s.MemoryLimit),
			s.MemoryPercentage,
			units.HumanSize(s.NetworkRx), units.HumanSize(s.NetworkTx),
			units.HumanSize(s.BlockRead), units.HumanSize(s.BlockWrite)))
	if processList != nil {
		topRenderer := NewDockerTopRenderer(processList)
		io.WriteString(w, topRenderer.Render())
	}
	w.Flush()
	return buf.String()
}

//NewDockerStatsBufferer creates termui bufferer for docker stats
func NewDockerStatsBufferer(stats *drydocker.Stats, x, y, height, width int) []termui.Bufferer {
	var result []termui.Bufferer
	top, length := NewDockerTopBufferer(stats.ProcessList, 0, y, height, width)
	result = append(result,
		top)
	yPos := y + length

	buf := bytes.NewBufferString("")

	w := tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)
	io.WriteString(w, "[%CPU\tMEM USAGE / LIMIT\t%MEM\tNET I/O\tBLOCK I/O](fg-blue)\n")
	io.WriteString(
		w,
		fmt.Sprintf("[%.2f\t%s / %s\t%.2f\t%s / %s\t%s / %s](fg-white)\n",
			stats.CPUPercentage,
			units.HumanSize(stats.Memory), units.HumanSize(stats.MemoryLimit),
			stats.MemoryPercentage,
			units.HumanSize(stats.NetworkRx), units.HumanSize(stats.NetworkTx),
			units.HumanSize(stats.BlockRead), units.HumanSize(stats.BlockWrite)))
	w.Flush()
	p := termui.NewPar(buf.String())
	p.X = x
	p.Y = yPos
	p.Height = height
	p.Width = width
	p.TextFgColor = termui.Attribute(termbox.ColorYellow)
	p.BorderLabel = " STATS "
	p.BorderLabelFg = termui.Attribute(termbox.ColorYellow)
	p.Border = true
	p.BorderBottom = false
	p.BorderLeft = false
	p.BorderRight = false
	p.BorderTop = true
	result = append(result, p)

	return result
}
