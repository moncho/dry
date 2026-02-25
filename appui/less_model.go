package appui

import (
	"regexp"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// CloseOverlayMsg signals that an overlay should be closed.
type CloseOverlayMsg struct{}

type lessMode int

const (
	lessNormal lessMode = iota
	lessSearching
	lessFiltering
)

// LessModel is a scrollable text viewer with search and filter support.
type LessModel struct {
	viewport    viewport.Model
	searchInput textinput.Model
	filterInput textinput.Model

	content   string   // original full content
	lines     []string // original lines
	filtered  []string // lines after filter
	mode      lessMode
	pattern   string // current search pattern
	filter    string // current filter pattern
	following bool   // auto-scroll to bottom
	title     string
	width     int
	height    int
}

// NewLessModel creates a new less viewer.
func NewLessModel() LessModel {
	vp := viewport.New()
	vp.SoftWrap = true
	// Customize keymap: remove "f" from PageDown since we use it for follow toggle
	km := viewport.DefaultKeyMap()
	km.PageDown = key.NewBinding(key.WithKeys("pgdown", "space"))
	vp.KeyMap = km

	si := textinput.New()
	si.Prompt = "/"
	si.Placeholder = "search..."
	si.CharLimit = 256

	fi := textinput.New()
	fi.Prompt = "Filter: "
	fi.Placeholder = "filter pattern..."
	fi.CharLimit = 256

	return LessModel{
		viewport:    vp,
		searchInput: si,
		filterInput: fi,
	}
}

// SetContent sets the text content to display.
func (m *LessModel) SetContent(content string, title string) {
	m.content = content
	m.title = title
	m.lines = strings.Split(content, "\n")
	m.filtered = m.lines
	m.filter = ""
	m.pattern = ""
	m.viewport.SetContent(content)
	m.viewport.ClearHighlights()
}

// AppendContent adds content (for streaming).
func (m *LessModel) AppendContent(text string) {
	m.content += text
	m.lines = strings.Split(m.content, "\n")
	if m.filter != "" {
		m.applyFilter()
	} else {
		m.filtered = m.lines
	}
	m.viewport.SetContent(strings.Join(m.filtered, "\n"))
	m.applySearch()
	if m.following {
		m.viewport.GotoBottom()
	}
}

// SetSize updates the viewport dimensions.
func (m *LessModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	// Reserve 1 line for title, 1 for status/input bar
	vpHeight := h - 2
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(vpHeight)
	// Re-apply content so viewport recalculates with new dimensions
	if m.content != "" {
		m.viewport.SetContent(strings.Join(m.filtered, "\n"))
		m.applySearch()
	}
}

// Update handles key events for the less viewer.
func (m LessModel) Update(msg tea.Msg) (LessModel, tea.Cmd) {
	switch m.mode {
	case lessSearching:
		return m.updateSearch(msg)
	case lessFiltering:
		return m.updateFilter(msg)
	default:
		return m.updateNormal(msg)
	}
}

func (m LessModel) updateNormal(msg tea.Msg) (LessModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return CloseOverlayMsg{} }
		case "/":
			m.mode = lessSearching
			m.searchInput.SetValue("")
			return m, m.searchInput.Focus()
		case "F":
			m.mode = lessFiltering
			m.filterInput.SetValue("")
			return m, m.filterInput.Focus()
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
		case "n":
			m.viewport.HighlightNext()
			return m, nil
		case "N":
			m.viewport.HighlightPrevious()
			return m, nil
		}
	}

	// Forward to viewport for scrolling (up/down/pgup/pgdown)
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m LessModel) updateSearch(msg tea.Msg) (LessModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.mode = lessNormal
			m.searchInput.Blur()
			return m, nil
		case "enter":
			m.mode = lessNormal
			m.pattern = m.searchInput.Value()
			m.searchInput.Blur()
			m.applySearch()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m LessModel) updateFilter(msg tea.Msg) (LessModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.mode = lessNormal
			m.filter = ""
			m.filtered = m.lines
			m.viewport.SetContent(strings.Join(m.filtered, "\n"))
			m.applySearch()
			m.filterInput.Blur()
			return m, nil
		case "enter":
			m.mode = lessNormal
			m.filter = m.filterInput.Value()
			m.filterInput.Blur()
			m.applyFilter()
			m.viewport.SetContent(strings.Join(m.filtered, "\n"))
			m.applySearch()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

func (m *LessModel) applySearch() {
	if m.pattern == "" {
		m.viewport.ClearHighlights()
		return
	}

	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(m.pattern))
	if err != nil {
		m.viewport.ClearHighlights()
		return
	}

	content := strings.Join(m.filtered, "\n")
	locs := re.FindAllStringIndex(content, -1)
	m.viewport.SetHighlights(locs)
	if len(locs) > 0 {
		m.viewport.HighlightNext()
	}
}

func (m *LessModel) applyFilter() {
	if m.filter == "" {
		m.filtered = m.lines
		return
	}
	lower := strings.ToLower(m.filter)
	m.filtered = nil
	for _, line := range m.lines {
		if strings.Contains(strings.ToLower(line), lower) {
			m.filtered = append(m.filtered, line)
		}
	}
}

// View renders the less viewer.
func (m LessModel) View() string {
	var sections []string

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(DryTheme.Fg).
		Background(DryTheme.Primary).
		Width(m.width)
	sections = append(sections, titleStyle.Render(m.title))

	// Viewport
	sections = append(sections, m.viewport.View())

	// Status/input bar
	switch m.mode {
	case lessSearching:
		sections = append(sections, m.searchInput.View())
	case lessFiltering:
		sections = append(sections, m.filterInput.View())
	default:
		status := m.statusLine()
		statusStyle := lipgloss.NewStyle().
			Foreground(DryTheme.FgSubtle).
			Width(m.width)
		sections = append(sections, statusStyle.Render(status))
	}

	return strings.Join(sections, "\n")
}

func (m LessModel) statusLine() string {
	parts := []string{"esc back"}
	if m.following {
		parts = append(parts, "f unfollow")
	} else {
		parts = append(parts, "f follow")
	}
	parts = append(parts, "/ search", "F filter")
	if m.pattern != "" {
		parts = append(parts, "n/N next/prev")
	}
	if m.filter != "" {
		parts = append(parts, "[filter: "+m.filter+"]")
	}
	return strings.Join(parts, "  ")
}
