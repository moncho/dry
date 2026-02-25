package appui

import (
	"fmt"
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
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(DryTheme.Key).Width(20)
	valueStyle := lipgloss.NewStyle().Foreground(DryTheme.Fg)

	title := titleStyle.Render("Docker Disk Usage")

	if m.usage == nil {
		return title + "\n\nLoading..."
	}

	du := m.usage

	// Calculate totals
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

	lines := []string{
		title,
		"",
		labelStyle.Render("Images:") + valueStyle.Render(
			fmt.Sprintf(" %d, Size: %s", len(du.Images), units.HumanSize(float64(imageSize)))),
		labelStyle.Render("Containers:") + valueStyle.Render(
			fmt.Sprintf(" %d, Size: %s", len(du.Containers), units.HumanSize(float64(containerSize)))),
		labelStyle.Render("Volumes:") + valueStyle.Render(
			fmt.Sprintf(" %d, Size: %s", len(du.Volumes), units.HumanSize(float64(volumeSize)))),
		labelStyle.Render("Build Cache:") + valueStyle.Render(
			fmt.Sprintf(" %d, Size: %s", len(du.BuildCache), units.HumanSize(float64(buildCacheSize)))),
		"",
		labelStyle.Render("Total:") + valueStyle.Render(
			fmt.Sprintf(" %s", units.HumanSize(float64(imageSize+containerSize+volumeSize+buildCacheSize)))),
	}

	// Pad to fill allocated height so the footer stays at the bottom.
	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	return strings.Join(lines, "\n")
}
