package appui

import (
	"fmt"

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

	// Label style
	label := lipgloss.NewStyle().Foreground(DryTheme.Key)
	// Value style
	value := lipgloss.NewStyle().Foreground(DryTheme.Fg)

	line1 := fmt.Sprintf("%s %s    %s %s    %s %s  %s %s",
		label.Render("Docker Host:"), value.Render(host),
		label.Render("Docker Version:"), value.Render(ver.Version),
		label.Render("Hostname:"), value.Render(info.Name),
		label.Render("Swarm:"), value.Render(swarmState),
	)

	line2 := fmt.Sprintf("%s %s    %s %s    %s %s",
		label.Render("Cert Path:"), value.Render(env.DockerCertPath),
		label.Render("APIVersion:"), value.Render(ver.APIVersion),
		label.Render("CPU:"), value.Render(fmt.Sprintf("%d", info.NCPU)),
	)

	line3 := fmt.Sprintf("%s %s    %s %s    %s %s",
		label.Render("Verify Certificate:"), value.Render(fmt.Sprintf("%t", env.DockerTLSVerify)),
		label.Render("OS/Arch/Kernel:"), value.Render(osArchKernel),
		label.Render("Memory:"), value.Render(memStr),
	)

	style := lipgloss.NewStyle().Width(m.width)
	return style.Render(line1 + "\n" + line2 + "\n" + line3)
}
