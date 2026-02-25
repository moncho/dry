package appui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"charm.land/lipgloss/v2"
	units "github.com/docker/go-units"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

// HeaderModel displays Docker daemon information at the top of the screen.
type HeaderModel struct {
	daemon docker.ContainerDaemon
	width  int
}

// NewHeaderModel creates a new header model.
func NewHeaderModel(daemon docker.ContainerDaemon, width int) HeaderModel {
	return HeaderModel{
		daemon: daemon,
		width:  width,
	}
}

// SetWidth updates the header width.
func (m *HeaderModel) SetWidth(w int) {
	m.width = w
}

// View renders the Docker daemon info header.
func (m HeaderModel) View() string {
	if m.daemon == nil {
		return ""
	}

	info, err := m.daemon.Info()
	if err != nil {
		return ui.Red("Error loading Docker info")
	}

	ver, err := m.daemon.Version()
	if err != nil {
		return ui.Red("Error loading Docker version")
	}

	env := m.daemon.DockerEnv()
	host := env.DockerHost
	if host == "" {
		host = docker.DefaultDockerHost
	}

	swarmState := string(info.Swarm.LocalNodeState)
	if swarmState == "" {
		swarmState = "inactive"
	}

	osArchKernel := fmt.Sprintf("%s/%s/%s", info.OSType, info.Architecture, info.KernelVersion)

	memStr := units.BytesSize(float64(info.MemTotal))

	// Label and value styles
	label := lipgloss.NewStyle().Foreground(DryTheme.Key)
	value := lipgloss.NewStyle().Foreground(DryTheme.Fg)

	// Fixed-width cells so columns align across all three lines.
	cellW1 := m.width * 38 / 100 // ~38% for column 1
	cellW2 := m.width * 32 / 100 // ~32% for column 2

	// renderCell pads or truncates styled content to exactly cellWidth visual chars.
	renderCell := func(l, v string, cellWidth int) string {
		content := label.Render(l) + value.Render(v)
		w := ansi.StringWidth(content)
		if w > cellWidth {
			return ansi.Truncate(content, cellWidth, "")
		}
		if w < cellWidth {
			return content + strings.Repeat(" ", cellWidth-w)
		}
		return content
	}

	line1 := renderCell("Docker Host: ", host, cellW1) +
		renderCell("Docker Version: ", ver.Version, cellW2) +
		label.Render("Hostname: ") + value.Render(info.Name) + "  " +
		label.Render("Swarm: ") + value.Render(swarmState)

	line2 := renderCell("Cert Path: ", env.DockerCertPath, cellW1) +
		renderCell("APIVersion: ", ver.APIVersion, cellW2) +
		label.Render("CPU: ") + value.Render(fmt.Sprintf("%d", info.NCPU))

	line3 := renderCell("Verify Certificate: ", fmt.Sprintf("%t", env.DockerTLSVerify), cellW1) +
		renderCell("OS/Arch/Kernel: ", osArchKernel, cellW2) +
		label.Render("Memory: ") + value.Render(memStr)

	// Pad each line to full width (truncate if overflow)
	line1 = padLine(line1, m.width)
	line2 = padLine(line2, m.width)
	line3 = padLine(line3, m.width)

	// Bottom border line
	borderLine := lipgloss.NewStyle().
		Background(DryTheme.Header).
		Width(m.width).
		Render(" ")

	return line1 + "\n" + line2 + "\n" + line3 + "\n" + borderLine
}

// padLine pads or truncates a line to exactly targetWidth visual characters.
func padLine(line string, targetWidth int) string {
	w := ansi.StringWidth(line)
	if w > targetWidth {
		return ansi.Truncate(line, targetWidth, "")
	}
	if w < targetWidth {
		return line + strings.Repeat(" ", targetWidth-w)
	}
	return line
}

// PadLine pads or truncates a line to exactly targetWidth visual characters,
// using the given style for padding spaces. This ensures backgrounds extend
// across the full width.
func PadLine(line string, targetWidth int, style lipgloss.Style) string {
	w := ansi.StringWidth(line)
	if w > targetWidth {
		return ansi.Truncate(line, targetWidth, "")
	}
	if w < targetWidth {
		return line + style.Render(strings.Repeat(" ", targetWidth-w))
	}
	return line
}
