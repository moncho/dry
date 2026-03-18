package appui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// InputPromptResultMsg carries the result of an input prompt.
type InputPromptResultMsg struct {
	Value     string
	Cancelled bool
	Tag       string // identifies which operation was being prompted
	ID        string // the resource ID being operated on
}

// InputPromptModel is a text input prompt overlay rendered as a centered floating window.
type InputPromptModel struct {
	message string
	tag     string
	id      string
	input   textinput.Model
	width   int
	height  int
}

// NewInputPromptModel creates a new input prompt.
func NewInputPromptModel(message, placeholder, tag, id string) (InputPromptModel, tea.Cmd) {
	return NewInputPromptModelWithLimit(message, placeholder, tag, id, 20)
}

// NewInputPromptModelWithLimit creates a new input prompt with a custom character limit.
func NewInputPromptModelWithLimit(message, placeholder, tag, id string, charLimit int) (InputPromptModel, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	cmd := ti.Focus()
	return InputPromptModel{
		message: message,
		tag:     tag,
		id:      id,
		input:   ti,
	}, cmd
}

// SetSize sets the overall screen size for centering the dialog.
func (m *InputPromptModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetWidth sets the prompt width (kept for backwards compatibility).
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

// View renders the input prompt as a centered floating dialog.
func (m InputPromptModel) View() string {
	dialogWidth := min(60, m.width-4)
	if dialogWidth < 20 {
		dialogWidth = 20
	}

	m.input.SetWidth(dialogWidth - 4) // account for padding/border

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(DryTheme.Fg).
		Width(dialogWidth - 4).
		Render(m.message)

	inputView := m.input.View()

	hint := lipgloss.NewStyle().
		Foreground(DryTheme.FgMuted).
		Render("Enter to confirm · Esc to cancel")

	body := lipgloss.JoinVertical(lipgloss.Left, title, "", inputView, "", hint)

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DryTheme.Primary).
		Padding(1, 2).
		Width(dialogWidth).
		Render(body)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}
