package appui

import (
	"fmt"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

// DockerEnvProvider is what the header needs from the daemon directly: the
// connection environment. Info and version arrive via SetDockerInfo.
type DockerEnvProvider interface {
	DockerEnv() docker.Env
}

// HeaderModel displays Docker daemon information at the top of the screen.
// The daemon info is loaded asynchronously via SetDockerInfo so constructing
// or rendering the header never performs I/O on the Update goroutine.
type HeaderModel struct {
	daemon DockerEnvProvider
	width  int

	// Cached Docker info, delivered by SetDockerInfo.
	loaded  bool
	info    system.Info
	ver     *client.ServerVersionResult
	infoErr error
	verErr  error
}

// NewHeaderModel creates a new header model. It performs no daemon calls;
// deliver the results of daemon.Info and daemon.Version via SetDockerInfo.
func NewHeaderModel(daemon DockerEnvProvider, width int) HeaderModel {
	return HeaderModel{
		daemon: daemon,
		width:  width,
	}
}

// SetWidth updates the header width.
func (m *HeaderModel) SetWidth(w int) {
	m.width = w
}

// SetDockerInfo stores the asynchronously fetched daemon info and version.
func (m *HeaderModel) SetDockerInfo(info system.Info, infoErr error, ver *client.ServerVersionResult, verErr error) {
	m.info = info
	m.infoErr = infoErr
	m.ver = ver
	m.verErr = verErr
	m.loaded = true
}

// View renders the Docker daemon info header.
func (m HeaderModel) View() string {
	if m.daemon == nil {
		return ""
	}

	if !m.loaded {
		// Keep the header height stable while the info loads.
		placeholder := lipgloss.NewStyle().Foreground(DryTheme.FgMuted).Render("Loading Docker daemon information...")
		return padLine(placeholder, m.width) + "\n" + padLine("", m.width) + "\n" + padLine("", m.width)
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

	// renderCell pads or truncates styled content to exactly cellWidth visual
	// chars, always reserving a one-column gap so adjacent columns never butt
	// together at narrow widths.
	const cellGap = 1
	renderCell := func(l, v string, cellWidth int) string {
		if cellWidth <= cellGap {
			return strings.Repeat(" ", max(cellWidth, 0))
		}
		styledLabel := label.Render(l)
		content := styledLabel + value.Render(v)
		w := ansi.StringWidth(content)
		if budget := cellWidth - cellGap; w > budget {
			// Spend a column on the ellipsis only when at least one value
			// character survives it; otherwise a plain cut shows more.
			tail := "…"
			if budget <= ansi.StringWidth(styledLabel)+1 {
				tail = ""
			}
			content = ansi.Truncate(content, budget, tail)
			w = ansi.StringWidth(content)
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
// The separator is one line in the layout's height budget, so a message is
// flattened and truncated first: a newline would make lipgloss render two
// rows and push the footer off the screen, and anything wider than the
// terminal would wrap for the same effect.
func (m HeaderModel) SeparatorLine(message string) string {
	if message != "" {
		message = oneLine(message)
		if m.width > 0 {
			message = ansi.Truncate(message, m.width, "…")
		}
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

// oneLine flattens a message so it cannot occupy more than one row, and so
// the row it occupies is the one it appears to. Newlines become separators,
// since lipgloss renders them as extra rows. A carriage return or backspace
// it passes through untouched at zero width, so the padding arithmetic is
// satisfied and the terminal still repaints over the row; compose writes
// them for progress. A tab it expands to four cells while ansi.StringWidth
// counts none, so one space in its place keeps the width honest and the
// words apart.
func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", "; ")
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case r != ' ' && unicode.IsControl(r):
			return -1
		}
		return r
	}, s)
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
