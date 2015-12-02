package ui

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/docker/docker/pkg/units"
	mdocker "github.com/moncho/dry/docker"
)

type statsRenderer struct {
	stats *mdocker.Stats
}

//NewDockerStatsRenderer creates renderer for docker stats
func NewDockerStatsRenderer(stats *mdocker.Stats) Renderer {
	return &statsRenderer{
		stats: stats,
	}
}

func (r *statsRenderer) Render() string {
	s := r.stats
	buf := bytes.NewBufferString("")
	w := tabwriter.NewWriter(buf, 22, 0, 1, ' ', 0)
	io.WriteString(w, "<green>CONTAINER\tCPU %\tMEM USAGE / LIMIT\tMEM %\tNET I/O\tBLOCK I/O</>\n")
	io.WriteString(
		w,
		fmt.Sprintf("<white>%s\t%.2f%%\t%s / %s\t%.2f%%\t%s / %s\t%s / %s</>\n",
			s.CID,
			s.CPUPercentage,
			units.HumanSize(s.Memory), units.HumanSize(s.MemoryLimit),
			s.MemoryPercentage,
			units.HumanSize(s.NetworkRx), units.HumanSize(s.NetworkTx),
			units.HumanSize(s.BlockRead), units.HumanSize(s.BlockWrite)))
	w.Flush()
	return buf.String()
}
