package appui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// FilterInputModel is a text input for filtering table rows.
type FilterInputModel struct {
	input  textinput.Model
	active bool
	width  int
}

// NewFilterInputModel creates a new filter input.
func NewFilterInputModel() FilterInputModel {
	ti := textinput.New()
	ti.Prompt = "Filter: "
	ti.Placeholder = "type to filter..."
	ti.CharLimit = 256
	return FilterInputModel{input: ti}
}

// Active returns whether the filter input is showing.
func (m FilterInputModel) Active() bool {
	return m.active
}

// Value returns the current filter text.
func (m FilterInputModel) Value() string {
	return m.input.Value()
}

// SetWidth sets the input width.
func (m *FilterInputModel) SetWidth(w int) {
	m.width = w
}

// Activate shows and focuses the filter input.
func (m *FilterInputModel) Activate() tea.Cmd {
	m.active = true
	m.input.SetValue("")
	return m.input.Focus()
}

// Deactivate hides the filter input and returns the final value.
func (m *FilterInputModel) Deactivate() string {
	m.active = false
	m.input.Blur()
	return m.input.Value()
}

// Clear resets and hides the filter input.
func (m *FilterInputModel) Clear() {
	m.active = false
	m.input.SetValue("")
	m.input.Blur()
}

// Update handles input events when active.
func (m FilterInputModel) Update(msg tea.Msg) (FilterInputModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.Clear()
			return m, nil
		case "enter":
			m.active = false
			m.input.Blur()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the filter input bar.
func (m FilterInputModel) View() string {
	if !m.active {
		return ""
	}
	style := lipgloss.NewStyle().Width(m.width)
	return style.Render(m.input.View())
}
