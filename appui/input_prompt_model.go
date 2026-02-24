package appui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// InputPromptResultMsg carries the result of an input prompt.
type InputPromptResultMsg struct {
	Value     string
	Cancelled bool
	Tag       string // identifies which operation was being prompted
	ID        string // the resource ID being operated on
}

// InputPromptModel is a text input prompt overlay.
type InputPromptModel struct {
	message string
	tag     string
	id      string
	input   textinput.Model
	width   int
}

// NewInputPromptModel creates a new input prompt.
func NewInputPromptModel(message, placeholder, tag, id string) (InputPromptModel, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 20
	ti.SetWidth(20)
	cmd := ti.Focus()
	return InputPromptModel{
		message: message,
		tag:     tag,
		id:      id,
		input:   ti,
	}, cmd
}

// SetWidth sets the prompt width.
func (m *InputPromptModel) SetWidth(w int) {
	m.width = w
}

// Update handles key events for the input prompt.
func (m InputPromptModel) Update(msg tea.Msg) (InputPromptModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			return m, func() tea.Msg {
				return InputPromptResultMsg{
					Value: m.input.Value(),
					Tag:   m.tag,
					ID:    m.id,
				}
			}
		case "esc":
			return m, func() tea.Msg {
				return InputPromptResultMsg{
					Cancelled: true,
					Tag:       m.tag,
					ID:        m.id,
				}
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the input prompt.
func (m InputPromptModel) View() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("25")).
		Width(m.width).
		Padding(0, 1)
	return style.Render(m.message + " " + m.input.View())
}
