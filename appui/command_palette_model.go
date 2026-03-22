package appui

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// CommandPaletteItem is a selectable command palette entry.
type CommandPaletteItem struct {
	ID          string
	Group       string
	Title       string
	Description string
	Search      string
}

// CommandPaletteResultMsg is emitted when a palette action is selected.
type CommandPaletteResultMsg struct {
	ActionID string
}

// CommandPaletteModel renders a centered command palette with filtering.
type CommandPaletteModel struct {
	input    textinput.Model
	items    []CommandPaletteItem
	filtered []CommandPaletteItem
	cursor   int
	width    int
	height   int
}

// NewCommandPaletteModel creates a command palette overlay.
func NewCommandPaletteModel(items []CommandPaletteItem) (CommandPaletteModel, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "Type a command"
	ti.Prompt = "> "
	ti.CharLimit = 120
	cmd := ti.Focus()

	m := CommandPaletteModel{
		input: ti,
		items: append([]CommandPaletteItem(nil), items...),
	}
	m.applyFilter()
	return m, cmd
}

// SetSize sets the screen size used to center the palette.
func (m *CommandPaletteModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Update handles keyboard input.
func (m CommandPaletteModel) Update(msg tea.Msg) (CommandPaletteModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CloseOverlayMsg{} }
		case "q":
			if strings.TrimSpace(m.input.Value()) == "" {
				return m, func() tea.Msg { return CloseOverlayMsg{} }
			}
		case "enter":
			if len(m.filtered) == 0 {
				return m, nil
			}
			item := m.filtered[m.cursor]
			return m, func() tea.Msg {
				return CommandPaletteResultMsg{ActionID: item.ID}
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	before := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	if before != m.input.Value() {
		m.applyFilter()
	}
	return m, cmd
}

// View renders the command palette overlay.
func (m CommandPaletteModel) View() string {
	dialogWidth := min(84, m.width-4)
	if m.width > 0 && dialogWidth > m.width {
		dialogWidth = m.width
	}
	if dialogWidth < 24 && m.width >= 24 {
		dialogWidth = 24
	}
	if dialogWidth < 16 {
		dialogWidth = max(m.width, 1)
	}
	if dialogWidth < 32 && m.width >= 32 {
		dialogWidth = 32
	}
	bodyWidth := dialogWidth - 6
	if bodyWidth < 8 {
		bodyWidth = max(dialogWidth-2, 1)
	}
	m.input.SetWidth(bodyWidth)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(DryTheme.Fg).
		Render("Command Palette")

	subtitle := lipgloss.NewStyle().
		Foreground(DryTheme.FgMuted).
		Render("Type to filter commands for the current context.")

	itemsHeight := m.height - 12
	if itemsHeight < 4 {
		itemsHeight = 4
	}
	if itemsHeight > 10 {
		itemsHeight = 10
	}

	listView := m.renderItems(bodyWidth, itemsHeight)
	hint := lipgloss.NewStyle().
		Foreground(DryTheme.FgSubtle).
		Render(ansi.Truncate("Enter run · Esc cancel · ↑↓ move", bodyWidth, "…"))

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		m.input.View(),
		"",
		listView,
		"",
		hint,
	)

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(DryTheme.Info).
		Padding(1, 2).
		Width(dialogWidth).
		Render(body)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}

func (m *CommandPaletteModel) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if query == "" {
		m.filtered = append([]CommandPaletteItem(nil), m.items...)
		m.cursor = min(m.cursor, max(len(m.filtered)-1, 0))
		return
	}

	tokens := strings.Fields(query)
	type scoredItem struct {
		item  CommandPaletteItem
		score int
	}
	matches := make([]scoredItem, 0, len(m.items))
	for _, item := range m.items {
		score := 0
		matched := true
		for _, token := range tokens {
			tokenScore := scorePaletteToken(item, token)
			if tokenScore == 0 {
				matched = false
				break
			}
			score += tokenScore
		}
		if matched {
			matches = append(matches, scoredItem{item: item, score: score})
		}
	}
	if len(matches) == 0 {
		m.filtered = nil
		m.cursor = 0
		return
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			if matches[i].item.Group == matches[j].item.Group {
				return matches[i].item.Title < matches[j].item.Title
			}
			return matches[i].item.Group < matches[j].item.Group
		}
		return matches[i].score > matches[j].score
	})
	m.filtered = m.filtered[:0]
	for _, match := range matches {
		m.filtered = append(m.filtered, match.item)
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

func (m CommandPaletteModel) renderItems(width, maxItems int) string {
	if len(m.filtered) == 0 {
		return lipgloss.NewStyle().
			Foreground(DryTheme.FgMuted).
			Width(width).
			Render("No matching commands.")
	}

	start := 0
	if m.cursor >= maxItems {
		start = m.cursor - maxItems + 1
	}
	end := start + maxItems
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	selectedStyle := lipgloss.NewStyle().
		Foreground(DryTheme.Bg).
		Background(DryTheme.Info).
		Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(DryTheme.Fg)
	descStyle := lipgloss.NewStyle().Foreground(DryTheme.FgMuted)

	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		item := m.filtered[i]
		line := item.Title
		if item.Description != "" {
			line += " · " + item.Description
		}
		if item.Group != "" {
			line = "[" + item.Group + "] " + line
		}
		line = ansi.Truncate(line, width-2, "…")
		if i == m.cursor {
			lines = append(lines, selectedStyle.Width(width).Padding(0, 1).Render(line))
			continue
		}
		groupStyle := lipgloss.NewStyle().
			Foreground(DryTheme.Info).
			Background(DryTheme.Header).
			Padding(0, 1)
		title := item.Title
		if item.Group != "" {
			title = groupStyle.Render(item.Group) + " " + title
		}
		if item.Description != "" {
			title = normalStyle.Render(item.Title) + descStyle.Render(" · "+item.Description)
			if item.Group != "" {
				title = groupStyle.Render(item.Group) + " " + normalStyle.Render(item.Title) + descStyle.Render(" · "+item.Description)
			}
		} else {
			if item.Group == "" {
				title = normalStyle.Render(item.Title)
			}
		}
		lines = append(lines, lipgloss.NewStyle().Width(width).Padding(0, 1).Render(title))
	}
	return strings.Join(lines, "\n")
}

func scorePaletteToken(item CommandPaletteItem, token string) int {
	title := strings.ToLower(item.Title)
	group := strings.ToLower(item.Group)
	desc := strings.ToLower(item.Description)
	search := strings.ToLower(item.Search)
	joined := strings.TrimSpace(strings.Join([]string{group, title, desc, search}, " "))

	switch {
	case token == title, token == group:
		return 180
	case strings.HasPrefix(title, token):
		return 140
	case strings.HasPrefix(group, token):
		return 130
	case containsWordPrefix(title, token):
		return 110
	case containsWordPrefix(group, token):
		return 100
	case strings.Contains(title, token):
		return 80
	case strings.Contains(group, token):
		return 75
	case strings.Contains(desc, token):
		return 50
	case strings.Contains(search, token):
		return 45
	}

	if score, ok := subsequenceScore(title, token); ok {
		return score + 40
	}
	if score, ok := subsequenceScore(group, token); ok {
		return score + 35
	}
	if score, ok := subsequenceScore(joined, token); ok {
		return score + 20
	}
	return 0
}

func containsWordPrefix(text, token string) bool {
	for _, part := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == ':' || r == '/' || r == '-' || r == '_'
	}) {
		if strings.HasPrefix(part, token) {
			return true
		}
	}
	return false
}

func subsequenceScore(text, token string) (int, bool) {
	if token == "" {
		return 0, false
	}
	ti := 0
	lastMatch := -1
	score := 0
	for i := 0; i < len(text) && ti < len(token); i++ {
		if text[i] != token[ti] {
			continue
		}
		score += 12
		if lastMatch >= 0 && i == lastMatch+1 {
			score += 6
		}
		lastMatch = i
		ti++
	}
	if ti != len(token) {
		return 0, false
	}
	if len(text) > 0 {
		score -= len(text) - len(token)
	}
	return max(score, 1), true
}
