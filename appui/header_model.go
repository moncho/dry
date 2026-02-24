package appui

import (
	"fmt"

	"charm.land/lipgloss/v2"
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

	host := m.daemon.DockerEnv().DockerHost
	if host == "" {
		host = docker.DefaultDockerHost
	}

	line1 := fmt.Sprintf(
		"%s | %s | Containers: %d (Running: %d) | Images: %d",
		ui.Blue("Docker Host: ")+ui.White(host),
		ui.Blue("Docker Version: ")+ui.White(ver.Version),
		info.Containers, info.ContainersRunning,
		info.Images,
	)

	line2 := fmt.Sprintf(
		"%s%s%s",
		ui.Blue("OS: ")+ui.White(info.OperatingSystem),
		"  ",
		ui.Blue("Kernel: ")+ui.White(info.KernelVersion),
	)

	style := lipgloss.NewStyle().Width(m.width)
	return style.Render(line1 + "\n" + line2)
}
