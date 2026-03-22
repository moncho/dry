package workspace

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/moncho/dry/appui"
)

// ActivityModel renders the bottom activity pane.
type ActivityModel struct {
	viewport  viewport.Model
	width     int
	height    int
	title     string
	status    string
	following bool
	focused   bool
	content   string
}

// NewActivityModel creates a new workspace activity pane.
func NewActivityModel() ActivityModel {
	vp := viewport.New()
	vp.SoftWrap = true
	return ActivityModel{
		viewport:  vp,
		following: true,
		title:     "Activity",
		status:    "Idle",
		content:   "Select an item or pin a log source to populate activity.",
	}
}

// SetSize updates the viewport dimensions.
func (m *ActivityModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	bodyHeight := h - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(bodyHeight)
	m.viewport.SetContent(m.content)
	if m.following {
		m.viewport.GotoBottom()
	}
}

// SetFocused toggles focus styling.
func (m *ActivityModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetContent replaces the displayed content.
func (m *ActivityModel) SetContent(title, status, content string) {
	m.title = title
	m.status = status
	m.content = content
	m.viewport.SetContent(content)
	if m.following {
		m.viewport.GotoBottom()
	}
}

// AppendContent appends streamed content.
func (m *ActivityModel) AppendContent(content string) {
	m.content += content
	offset := m.viewport.YOffset()
	m.viewport.SetContent(m.content)
	if m.following {
		m.viewport.GotoBottom()
	} else {
		m.viewport.SetYOffset(offset)
	}
}

// Clear resets the pane to a placeholder.
func (m *ActivityModel) Clear(title, status, content string) {
	m.title = title
	m.status = status
	m.following = true
	m.SetContent(m.title, m.status, content)
}

// Width returns the current pane width.
func (m ActivityModel) Width() int {
	return m.width
}

// Height returns the current pane height.
func (m ActivityModel) Height() int {
	return m.height
}

// BodyHeight returns the usable content height below the header and status line.
func (m ActivityModel) BodyHeight() int {
	bodyHeight := m.height - 2
	if bodyHeight < 1 {
		return 1
	}
	return bodyHeight
}

// Update handles pane-local navigation.
func (m ActivityModel) Update(msg tea.Msg) (ActivityModel, tea.Cmd) {
	if m.isStaticMonitorDetails() {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f":
			m.following = !m.following
			if m.following {
				m.viewport.GotoBottom()
			}
			return m, nil
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

// View renders the activity pane.
func (m ActivityModel) View() string {
	accent := appui.DryTheme.Info
	headerBg := appui.DryTheme.Header
	if m.focused {
		accent = appui.DryTheme.Bg
		headerBg = appui.DryTheme.Info
	}
	title := m.title
	total := len(strings.Split(m.content, "\n"))
	filtered := total
	if m.following && !m.hideFollowInHeader() {
		title += " · follow"
	}
	if m.hideCountsInHeader() {
		total = -1
		filtered = -1
	}
	header := appui.RenderWidgetHeader(appui.WidgetHeaderOpts{
		Icon:     "≋",
		Title:    title,
		Total:    total,
		Filtered: filtered,
		Width:    m.width,
		Accent:   accent,
		HeaderBg: headerBg,
	})
	bodyHeight := m.BodyHeight()
	status := lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Foreground(appui.DryTheme.FgSubtle).
		Render(m.status)
	bodyContent := m.viewport.View()
	if m.isStaticMonitorDetails() {
		bodyContent = m.content
	}
	body := lipgloss.NewStyle().
		Width(m.width).
		Height(bodyHeight).
		Render(bodyContent)
	return header + status + "\n" + body
}

func (m ActivityModel) hideFollowInHeader() bool {
	return strings.HasPrefix(m.title, "Monitor Details")
}

func (m ActivityModel) hideCountsInHeader() bool {
	return strings.HasPrefix(m.title, "Monitor Details")
}

func (m ActivityModel) isStaticMonitorDetails() bool {
	return strings.HasPrefix(m.title, "Monitor Details")
}
