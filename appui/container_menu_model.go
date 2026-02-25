package appui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
)

// ContainerMenuCommandMsg is sent when a menu command is selected.
type ContainerMenuCommandMsg struct {
	ContainerID string
	Command     docker.Command
}

// ContainerMenuModel shows a command menu for a selected container.
type ContainerMenuModel struct {
	containerID string
	commands    []docker.CommandDescription
	cursor      int
	width       int
	height      int
	infoLines   []string
}

// NewContainerMenuModel creates a container menu for the given container.
func NewContainerMenuModel(c *docker.Container) ContainerMenuModel {
	cf := formatter.NewContainerFormatter(c, true)
	info := []string{
		fmt.Sprintf("Container: %s", cf.Names()),
		fmt.Sprintf("Image:     %s", cf.Image()),
		fmt.Sprintf("Status:    %s", cf.Status()),
		fmt.Sprintf("ID:        %s", cf.ID()),
	}
	return ContainerMenuModel{
		containerID: c.ID,
		commands:    docker.ContainerCommands,
		infoLines:   info,
	}
}

// SetSize updates the menu dimensions.
func (m *ContainerMenuModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Update handles key events for the menu.
func (m ContainerMenuModel) Update(msg tea.Msg) (ContainerMenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return CloseOverlayMsg{} }
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.commands) - 1
			}
			return m, nil
		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.commands) {
				m.cursor = 0
			}
			return m, nil
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.commands) {
				cmd := m.commands[m.cursor]
				return m, func() tea.Msg {
					return ContainerMenuCommandMsg{
						ContainerID: m.containerID,
						Command:     cmd.Command,
					}
				}
			}
			return m, nil
		}
	}
	return m, nil
}

// View renders the container menu.
func (m ContainerMenuModel) View() string {
	menuWidth := 40

	// Container info section
	infoStyle := lipgloss.NewStyle().
		Foreground(Ash).
		Bold(true)

	var sections []string
	for _, line := range m.infoLines {
		sections = append(sections, infoStyle.Render(line))
	}
	sections = append(sections, "") // blank separator

	// Menu items
	normalStyle := lipgloss.NewStyle().
		Width(menuWidth).
		Padding(0, 1)
	selectedStyle := lipgloss.NewStyle().
		Width(menuWidth).
		Padding(0, 1).
		Background(Charple).
		Foreground(Ash)

	for i, cmd := range m.commands {
		style := normalStyle
		if i == m.cursor {
			style = selectedStyle
		}
		sections = append(sections, style.Render(cmd.Description))
	}

	sections = append(sections, "")
	hintStyle := lipgloss.NewStyle().Foreground(Squid)
	sections = append(sections, hintStyle.Render("ESC:back  Enter:execute"))

	menu := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, menu)
}
