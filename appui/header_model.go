package appui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	units "github.com/docker/go-units"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/system"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

// HeaderModel displays Docker daemon information at the top of the screen.
type HeaderModel struct {
	daemon docker.ContainerDaemon
	width  int

	// Cached Docker info
	info    system.Info
	ver     *dockerTypes.Version
	infoErr error
	verErr  error
}

// NewHeaderModel creates a new header model.
func NewHeaderModel(daemon docker.ContainerDaemon, width int) HeaderModel {
	m := HeaderModel{
		daemon: daemon,
		width:  width,
	}
	m.Refresh()
	return m
}

// SetWidth updates the header width.
func (m *HeaderModel) SetWidth(w int) {
	m.width = w
}

// Refresh re-fetches Docker info and version from the daemon.
func (m *HeaderModel) Refresh() {
	if m.daemon == nil {
		return
	}
	m.info, m.infoErr = m.daemon.Info()
	m.ver, m.verErr = m.daemon.Version()
}

// View renders the Docker daemon info header.
func (m HeaderModel) View() string {
	if m.daemon == nil {
		return ""
	}

	if m.infoErr != nil {
		return ui.Red("Error loading Docker info")
	}
	if m.verErr != nil || m.ver == nil {
		return ui.Red("Error loading Docker version")
	}

	env := m.daemon.DockerEnv()
	host := env.DockerHost
	if host == "" {
		host = docker.DefaultDockerHost
	}

	swarmState := string(m.info.Swarm.LocalNodeState)
	if swarmState == "" {
		swarmState = "inactive"
	}

	osArchKernel := fmt.Sprintf("%s/%s/%s", m.info.OSType, m.info.Architecture, m.info.KernelVersion)

	memStr := units.BytesSize(float64(m.info.MemTotal))

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
		renderCell("Docker Version: ", m.ver.Version, cellW2) +
		label.Render("Hostname: ") + value.Render(m.info.Name) + "  " +
		label.Render("Swarm: ") + value.Render(swarmState)

	line2 := renderCell("Cert Path: ", env.DockerCertPath, cellW1) +
		renderCell("APIVersion: ", m.ver.APIVersion, cellW2) +
		label.Render("CPU: ") + value.Render(fmt.Sprintf("%d", m.info.NCPU))

	line3 := renderCell("Verify Certificate: ", fmt.Sprintf("%t", env.DockerTLSVerify), cellW1) +
		renderCell("OS/Arch/Kernel: ", osArchKernel, cellW2) +
		label.Render("Memory: ") + value.Render(memStr)

	// Pad each line to full width (truncate if overflow)
	line1 = padLine(line1, m.width)
	line2 = padLine(line2, m.width)
	line3 = padLine(line3, m.width)

	return line1 + "\n" + line2 + "\n" + line3
}

// SeparatorLine renders the header separator. When message is non-empty it
// displays the message text; otherwise it renders a plain colored line.
func (m HeaderModel) SeparatorLine(message string) string {
	if message != "" {
		style := lipgloss.NewStyle().
			Foreground(DryTheme.Fg).
			Background(DryTheme.Header).
			Width(m.width).
			MaxWidth(m.width)
		return style.Render(message)
	}
	return lipgloss.NewStyle().
		Background(DryTheme.Header).
		Width(m.width).
		Render(" ")
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
