package appui

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/go-units"
	"github.com/moncho/dry/docker"
)

// DiskUsageLoadedMsg carries the loaded disk usage data.
type DiskUsageLoadedMsg struct {
	Usage types.DiskUsage
}

// DiskUsageModel displays Docker disk usage information.
type DiskUsageModel struct {
	usage  *types.DiskUsage
	daemon docker.ContainerDaemon
	width  int
	height int
}

// NewDiskUsageModel creates a disk usage model.
func NewDiskUsageModel() DiskUsageModel {
	return DiskUsageModel{}
}

// SetDaemon sets the Docker daemon reference.
func (m *DiskUsageModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// SetSize updates the dimensions.
func (m *DiskUsageModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetUsage replaces the disk usage data.
func (m *DiskUsageModel) SetUsage(usage types.DiskUsage) {
	m.usage = &usage
}

// Update handles key events.
func (m DiskUsageModel) Update(msg tea.Msg) (DiskUsageModel, tea.Cmd) {
	// No navigation needed — just display
	return m, nil
}

// View renders the disk usage summary.
func (m DiskUsageModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(DryTheme.Key)
	title := titleStyle.Render("Docker Disk Usage")

	if m.usage == nil {
		lines := []string{title, "", "Loading..."}
		for len(lines) < m.height {
			lines = append(lines, strings.Repeat(" ", m.width))
		}
		return strings.Join(lines, "\n")
	}

	du := m.usage

	// Calculate sizes per category
	var imageSize int64
	for _, img := range du.Images {
		imageSize += img.Size
	}
	var containerSize int64
	for _, c := range du.Containers {
		containerSize += c.SizeRw
	}
	var volumeSize int64
	for _, v := range du.Volumes {
		volumeSize += v.UsageData.Size
	}
	var buildCacheSize int64
	for _, bc := range du.BuildCache {
		buildCacheSize += bc.Size
	}

	total := imageSize + containerSize + volumeSize + buildCacheSize

	barWidth := 30
	if m.width > 80 {
		barWidth = 40
	}

	type category struct {
		label string
		count int
		size  int64
		color color.Color
	}
	cats := []category{
		{"Images", len(du.Images), imageSize, DryTheme.Tertiary},
		{"Containers", len(du.Containers), containerSize, DryTheme.Secondary},
		{"Volumes", len(du.Volumes), volumeSize, DryTheme.Info},
		{"Build Cache", len(du.BuildCache), buildCacheSize, DryTheme.Warning},
	}

	labelWidth := 14
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(DryTheme.Key).Width(labelWidth)
	valueStyle := lipgloss.NewStyle().Foreground(DryTheme.Fg)
	totalStyle := lipgloss.NewStyle().Bold(true).Foreground(DryTheme.Fg)

	lines := []string{title, ""}

	for _, cat := range cats {
		label := labelStyle.Render(cat.label)
		info := valueStyle.Render(fmt.Sprintf(" %3d   %s", cat.count, units.HumanSize(float64(cat.size))))
		bar := renderBar(cat.size, total, barWidth, cat.color, DryTheme.Border)
		lines = append(lines, label+info+"  "+bar)
	}

	lines = append(lines, "")
	totalLabel := lipgloss.NewStyle().Bold(true).Foreground(DryTheme.Key).Width(labelWidth).Render("Total")
	totalBar := renderBar(total, total, barWidth, DryTheme.Primary, DryTheme.Border)
	lines = append(lines, totalLabel+totalStyle.Render(fmt.Sprintf("       %s", units.HumanSize(float64(total))))+"  "+totalBar)

	// Pad to fill allocated height so the footer stays at the bottom.
	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	return strings.Join(lines, "\n")
}

// renderBar creates a horizontal bar chart segment.
func renderBar(value, total int64, width int, fg, bg color.Color) string {
	if total <= 0 || width <= 0 {
		bgStyle := lipgloss.NewStyle().Foreground(bg)
		return bgStyle.Render(strings.Repeat("─", width))
	}

	filled := int(float64(value) * float64(width) / float64(total))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	empty := width - filled

	fgStyle := lipgloss.NewStyle().Foreground(fg)
	bgStyle := lipgloss.NewStyle().Foreground(bg)

	return fgStyle.Render(strings.Repeat("█", filled)) + bgStyle.Render(strings.Repeat("─", empty))
}
