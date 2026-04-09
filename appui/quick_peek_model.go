package appui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	quickPeekVerticalChrome   = 4
	quickPeekMinContentHeight = 2
	quickPeekFixedBodyLines   = 9
)

// QuickPeekModel renders a temporary side-panel preview for the current
// selection without leaving the active list.
type QuickPeekModel struct {
	viewport     viewport.Model
	screenWidth  int
	screenHeight int
	title        string
	subtitle     string
	detailTitle  string
	status       string
	summary      []string
	content      string
}

// NewQuickPeekModel creates a new quick peek overlay.
func NewQuickPeekModel() QuickPeekModel {
	vp := viewport.New()
	vp.SoftWrap = true
	return QuickPeekModel{
		viewport:    vp,
		detailTitle: "Preview",
		status:      "Loading preview...",
		content:     "Preparing quick peek...",
	}
}

// SetSize sets the screen dimensions used to place and size the overlay.
func (m *QuickPeekModel) SetSize(w, h int) {
	m.screenWidth = w
	m.screenHeight = h
	m.syncViewport()
}

// SetContent updates the preview content.
func (m *QuickPeekModel) SetContent(title, subtitle, detailTitle, status string, summary []string, content string) {
	m.title = title
	m.subtitle = subtitle
	m.detailTitle = detailTitle
	m.status = status
	m.summary = append([]string(nil), summary...)
	m.content = normalizeQuickPeekContent(content)
	m.syncViewport()
}

// Update handles overlay-local navigation.
func (m QuickPeekModel) Update(msg tea.Msg) (QuickPeekModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q", " ", "space":
			return m, func() tea.Msg { return CloseOverlayMsg{} }
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		case "j":
			msg.Code = tea.KeyDown
		case "k":
			msg.Code = tea.KeyUp
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the quick peek overlay.
func (m QuickPeekModel) View() string {
	dialogW, dialogH := m.dialogSize()
	bodyH := dialogH - quickPeekVerticalChrome
	if bodyH < 1 {
		bodyH = 1
	}
	innerW := max(dialogW-6, 1)

	header := strings.TrimRight(RenderWidgetHeader(WidgetHeaderOpts{
		Icon:     "◫",
		Title:    "Quick Peek",
		Total:    -1,
		Filtered: -1,
		Width:    innerW,
		Accent:   DryTheme.Info,
	}), "\n")

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(DryTheme.Fg).
		Render(m.title)
	subtitle := lipgloss.NewStyle().
		Foreground(DryTheme.FgMuted).
		Render(m.subtitle)
	status := lipgloss.NewStyle().
		Foreground(DryTheme.FgSubtle).
		Render(ansi.Truncate(m.status, innerW, "…"))
	hint := lipgloss.NewStyle().
		Foreground(DryTheme.FgSubtle).
		Render(ansi.Truncate("Space/Esc close · ↑↓ scroll · g/G jump", innerW, "…"))

	summaryLines := m.renderSummary(innerW, m.summaryLineLimit(bodyH))
	summaryBlock := lipgloss.NewStyle().
		Foreground(DryTheme.Fg).
		Width(innerW).
		Render(strings.Join(summaryLines, "\n"))
	divider := lipgloss.NewStyle().
		Foreground(DryTheme.FgSubtle).
		Render(strings.Repeat("─", innerW))

	detailTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(DryTheme.Info).
		Render(ansi.Truncate(m.detailTitle, innerW, "…"))

	contentHeight := bodyH - (len(summaryLines) + quickPeekFixedBodyLines)
	if contentHeight < quickPeekMinContentHeight {
		contentHeight = quickPeekMinContentHeight
	}
	m.viewport.SetWidth(innerW)
	m.viewport.SetHeight(contentHeight)
	m.viewport.SetContent(m.content)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		title,
		subtitle,
		"",
		summaryBlock,
		divider,
		detailTitle,
		status,
		m.viewport.View(),
		"",
		hint,
	)

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DryTheme.Info).
		Background(DryTheme.Bg).
		Padding(1, 2).
		Width(dialogW).
		Height(bodyH).
		Render(body)

	topMargin := m.topMargin()
	usableHeight := m.availableHeight()
	if usableHeight < 1 {
		usableHeight = 1
		topMargin = 0
	}
	placedLines := strings.Split(strings.TrimRight(lipgloss.Place(m.screenWidth, usableHeight, lipgloss.Center, lipgloss.Top, dialog), "\n"), "\n")
	if len(placedLines) > usableHeight {
		placedLines = placedLines[:usableHeight]
	}
	lines := make([]string, 0, m.screenHeight)
	blank := strings.Repeat(" ", max(m.screenWidth, 1))
	for i := 0; i < topMargin; i++ {
		lines = append(lines, blank)
	}
	lines = append(lines, placedLines...)
	for len(lines) < m.screenHeight {
		lines = append(lines, blank)
	}
	if len(lines) > m.screenHeight {
		lines = lines[:m.screenHeight]
	}
	return strings.Join(lines, "\n")
}

func (m *QuickPeekModel) syncViewport() {
	dialogW, dialogH := m.dialogSize()
	bodyH := dialogH - quickPeekVerticalChrome
	if bodyH < 1 {
		bodyH = 1
	}
	innerW := max(dialogW-6, 1)
	contentHeight := bodyH - (len(m.renderSummary(innerW, m.summaryLineLimit(bodyH))) + quickPeekFixedBodyLines)
	if contentHeight < quickPeekMinContentHeight {
		contentHeight = quickPeekMinContentHeight
	}
	m.viewport.SetWidth(innerW)
	m.viewport.SetHeight(contentHeight)
	m.viewport.SetContent(m.content)
	m.viewport.GotoTop()
}

func (m QuickPeekModel) dialogSize() (int, int) {
	width := m.screenWidth * 60 / 100
	if width < 58 {
		width = 58
	}
	if width > 104 {
		width = 104
	}
	if m.screenWidth > 0 && width > m.screenWidth {
		width = m.screenWidth
	}
	if width < 24 {
		width = max(m.screenWidth, 1)
	}

	height := m.availableHeight()
	if height < 1 {
		height = 1
	}
	return width, height
}

func (m QuickPeekModel) availableHeight() int {
	if m.screenHeight <= 4 {
		return max(m.screenHeight, 1)
	}
	return m.screenHeight - m.topMargin() - m.bottomMargin()
}

func (m QuickPeekModel) topMargin() int {
	if m.screenHeight < 18 {
		return 1
	}
	return 2
}

func (m QuickPeekModel) bottomMargin() int {
	return 1
}

func (m QuickPeekModel) summaryLineLimit(bodyH int) int {
	limit := 8
	if bodyH >= 30 {
		limit = 10
	} else if bodyH < 20 {
		limit = 6
	}
	maxSummary := bodyH - (quickPeekFixedBodyLines + quickPeekMinContentHeight)
	if maxSummary < 1 {
		maxSummary = 1
	}
	if limit > maxSummary {
		limit = maxSummary
	}
	return limit
}

func (m QuickPeekModel) renderSummary(width, maxLines int) []string {
	lines := make([]string, 0, len(m.summary)+1)
	for _, line := range m.summary {
		lines = append(lines, renderQuickPeekLine(line, width))
		if len(lines) == maxLines {
			break
		}
	}
	if len(m.summary) > len(lines) {
		lines[len(lines)-1] = lipgloss.NewStyle().Foreground(DryTheme.FgMuted).Render("...")
	}
	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(DryTheme.FgMuted).Render("No summary available."))
	}
	return lines
}

func renderQuickPeekLine(line string, width int) string {
	if line == "" {
		return line
	}
	if key, value, ok := strings.Cut(line, ": "); ok {
		line = lipgloss.NewStyle().Foreground(DryTheme.Key).Render(key+":") + " " +
			lipgloss.NewStyle().Foreground(DryTheme.Fg).Render(value)
	}
	return ansi.Truncate(line, width, "…")
}

func normalizeQuickPeekContent(content string) string {
	return strings.TrimRight(content, "\n")
}
