package appui

import (
	"fmt"

	"charm.land/bubbles/v2/list"
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

// commandItem wraps a docker.CommandDescription as a list.DefaultItem.
type commandItem struct {
	cmd docker.CommandDescription
}

func (i commandItem) Title() string       { return i.cmd.Description }
func (i commandItem) Description() string { return "" }
func (i commandItem) FilterValue() string { return i.cmd.Description }

// ContainerMenuModel shows a command menu for a selected container.
type ContainerMenuModel struct {
	containerID string
	header      string
	list        list.Model
	width       int
	height      int
}

// NewContainerMenuModel creates a container menu for the given container.
func NewContainerMenuModel(c *docker.Container) ContainerMenuModel {
	cf := formatter.NewContainerFormatter(c, true)

	items := make([]list.Item, len(docker.ContainerCommands))
	for i, cmd := range docker.ContainerCommands {
		items[i] = commandItem{cmd: cmd}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	delegate.Styles = list.DefaultItemStyles{
		NormalTitle: lipgloss.NewStyle().
			Foreground(DryTheme.Fg).
			Padding(0, 0, 0, 2),
		SelectedTitle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(DryTheme.Primary).
			Foreground(DryTheme.Fg).
			Padding(0, 0, 0, 1),
		DimmedTitle: lipgloss.NewStyle().
			Foreground(DryTheme.FgMuted).
			Padding(0, 0, 0, 2),
	}

	header := fmt.Sprintf("Container: %s\nImage:     %s\nStatus:    %s\nID:        %s",
		cf.Names(), cf.Image(), cf.Status(), cf.ID())

	// List height: just items (title is rendered separately).
	listHeight := len(items) + 2
	l := list.New(items, delegate, 50, listHeight)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.DisableQuitKeybindings()
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(DryTheme.FgMuted)

	return ContainerMenuModel{
		containerID: c.ID,
		header:      header,
		list:        l,
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
		case "enter":
			if item, ok := m.list.SelectedItem().(commandItem); ok {
				return m, func() tea.Msg {
					return ContainerMenuCommandMsg{
						ContainerID: m.containerID,
						Command:     item.cmd.Command,
					}
				}
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the container menu.
func (m ContainerMenuModel) View() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(DryTheme.FgMuted).
		Bold(true).
		MarginBottom(1)
	hintStyle := lipgloss.NewStyle().Foreground(DryTheme.FgSubtle)
	hint := hintStyle.Render("esc back · enter execute")

	inner := lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(m.header), m.list.View(), "", hint)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DryTheme.Border).
		Padding(1, 2)

	menu := box.Render(inner)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, menu)
}
