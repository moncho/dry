package appui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// PromptResultMsg carries the result of a prompt confirmation.
type PromptResultMsg struct {
	Confirmed bool
	Tag       string // identifies which operation was being prompted
	ID        string // the resource ID being operated on
}

// PromptModel is a y/N confirmation prompt overlay.
type PromptModel struct {
	message string
	tag     string
	id      string
	width   int
}

// NewPromptModel creates a new prompt.
func NewPromptModel(message, tag, id string) PromptModel {
	return PromptModel{
		message: message,
		tag:     tag,
		id:      id,
	}
}

// SetWidth sets the prompt width.
func (m *PromptModel) SetWidth(w int) {
	m.width = w
}

// Update handles key events for the prompt.
func (m PromptModel) Update(msg tea.Msg) (PromptModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "y", "Y":
			return m, func() tea.Msg {
				return PromptResultMsg{Confirmed: true, Tag: m.tag, ID: m.id}
			}
		case "n", "N", "esc", "enter":
			return m, func() tea.Msg {
				return PromptResultMsg{Confirmed: false, Tag: m.tag, ID: m.id}
			}
		}
	}
	return m, nil
}

// View renders the prompt.
func (m PromptModel) View() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("124")).
		Width(m.width).
		Padding(0, 1)
	return style.Render(m.message + " [y/N]")
}
