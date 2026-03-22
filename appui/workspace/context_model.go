package workspace

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/moncho/dry/appui"
)

// ContextModel renders the workspace summary pane.
type ContextModel struct {
	width    int
	height   int
	mode     string
	title    string
	subtitle string
	lines    []string
	empty    string
	focused  bool
	viewport viewport.Model
	content  string
}

// NewContextModel creates a workspace context pane.
func NewContextModel() ContextModel {
	vp := viewport.New()
	vp.SoftWrap = false
	return ContextModel{viewport: vp}
}

// SetSize updates the pane dimensions.
func (m *ContextModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	bodyHeight := h - 1
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	contentWidth := w - 2
	if contentWidth < 1 {
		contentWidth = 1
	}
	m.viewport.SetWidth(contentWidth)
	m.viewport.SetHeight(bodyHeight)
	m.viewport.SetContent(m.content)
}

// SetFocused toggles focus styling.
func (m *ContextModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetMode updates the context header mode label.
func (m *ContextModel) SetMode(mode string) {
	m.mode = mode
}

// SetContent replaces the pane content.
func (m *ContextModel) SetContent(title, subtitle string, lines []string) {
	if m.title == title && m.subtitle == subtitle && slices.Equal(m.lines, lines) {
		return
	}
	resetScroll := m.title != title || m.subtitle != subtitle
	m.title = title
	m.subtitle = subtitle
	m.lines = append([]string(nil), lines...)
	m.syncContent(resetScroll)
}

// SetEmptyMessage updates the empty-state message.
func (m *ContextModel) SetEmptyMessage(message string) {
	if m.empty == message {
		return
	}
	m.empty = message
	m.syncContent(true)
}

// Update handles pane-local navigation.
func (m ContextModel) Update(msg tea.Msg) (ContextModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the context pane.
func (m ContextModel) View() string {
	accent := appui.DryTheme.Info
	headerBg := appui.DryTheme.Header
	if m.focused {
		accent = appui.DryTheme.Bg
		headerBg = appui.DryTheme.Info
	}
	header := appui.RenderWidgetHeader(appui.WidgetHeaderOpts{
		Icon:     "◎",
		Title:    contextHeaderTitle(m.mode),
		Total:    -1,
		Filtered: -1,
		Width:    m.width,
		Accent:   accent,
		HeaderBg: headerBg,
	})

	bodyHeight := m.height - 1
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	body := lipgloss.NewStyle().
		Width(m.width).
		Height(bodyHeight).
		Padding(0, 1).
		Render(m.viewport.View())

	return header + body
}

func contextHeaderTitle(mode string) string {
	if mode == "" {
		return "Context"
	}
	return "Context · " + mode
}

func renderContextLine(line string) string {
	if line == "" || strings.Contains(line, "\x1b[") {
		return line
	}
	key, value, ok := strings.Cut(line, ": ")
	if !ok {
		return line
	}
	keyStyle := lipgloss.NewStyle().Foreground(appui.DryTheme.Key)
	valueStyle := lipgloss.NewStyle().Foreground(appui.DryTheme.Fg)
	return keyStyle.Render(key+":") + " " + valueStyle.Render(value)
}

func (m *ContextModel) syncContent(resetScroll bool) {
	offset := m.viewport.YOffset()
	bodyLines := make([]string, 0, len(m.lines)+2)
	if m.title != "" {
		bodyLines = append(bodyLines, lipgloss.NewStyle().Bold(true).Foreground(appui.DryTheme.Fg).Render(m.title))
	}
	if m.subtitle != "" {
		bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(appui.DryTheme.FgMuted).Render(m.subtitle))
	}
	if len(bodyLines) > 0 && len(m.lines) > 0 {
		bodyLines = append(bodyLines, "")
	}
	bodyLines = append(bodyLines, m.lines...)
	if len(bodyLines) == 0 {
		empty := m.empty
		if empty == "" {
			empty = "Select an item to preview it here."
		}
		bodyLines = append(bodyLines, lipgloss.NewStyle().Foreground(appui.DryTheme.FgMuted).Render(empty))
	}
	for i := range bodyLines {
		bodyLines[i] = renderContextLine(bodyLines[i])
	}
	contentWidth := m.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}
	for i := range bodyLines {
		bodyLines[i] = ansi.Truncate(bodyLines[i], contentWidth, "…")
	}
	m.content = strings.Join(bodyLines, "\n")
	m.viewport.SetContent(m.content)
	if resetScroll {
		m.viewport.GotoTop()
		return
	}
	m.viewport.SetYOffset(offset)
}
