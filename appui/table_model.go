package appui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Column defines a table column.
type Column struct {
	Title string
	Width int  // 0 means proportional (share remaining space)
	Fixed bool // fixed-width column
}

// TableRow represents one row of data in the table.
type TableRow interface {
	Columns() []string
	ID() string
}

// TableModel is a shared table component with navigation, sorting, and filtering.
type TableModel struct {
	columns     []Column
	rows        []TableRow
	filtered    []TableRow
	cursor      int
	offset      int // scroll offset for visible window
	width       int
	height      int
	sortField   int
	sortAsc     bool
	filterText  string
	filterFn    func(row TableRow, pattern string) bool
	colWidths   []int
}

// NewTableModel creates a table with the given column definitions.
func NewTableModel(columns []Column) TableModel {
	return TableModel{
		columns:  columns,
		sortAsc:  true,
		filterFn: defaultFilter,
	}
}

func defaultFilter(row TableRow, pattern string) bool {
	lower := strings.ToLower(pattern)
	for _, col := range row.Columns() {
		if strings.Contains(strings.ToLower(col), lower) {
			return true
		}
	}
	return false
}

// SetRows replaces all rows and reapplies the filter.
func (m *TableModel) SetRows(rows []TableRow) {
	m.rows = rows
	m.applyFilter()
	m.clampCursor()
}

// SetSize updates the table dimensions.
func (m *TableModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.calculateColumnWidths()
}

// SelectedRow returns the row under the cursor, or nil.
func (m TableModel) SelectedRow() TableRow {
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		return m.filtered[m.cursor]
	}
	return nil
}

// Cursor returns the current cursor position.
func (m TableModel) Cursor() int {
	return m.cursor
}

// RowCount returns the number of visible (filtered) rows.
func (m TableModel) RowCount() int {
	return len(m.filtered)
}

// TotalRowCount returns the total number of unfiltered rows.
func (m TableModel) TotalRowCount() int {
	return len(m.rows)
}

// FilterText returns the active filter string.
func (m TableModel) FilterText() string {
	return m.filterText
}

// SetFilter sets the filter text and reapplies it.
func (m *TableModel) SetFilter(pattern string) {
	m.filterText = pattern
	m.applyFilter()
	m.clampCursor()
}

// NextSort cycles to the next sort field.
func (m *TableModel) NextSort() {
	m.sortField = (m.sortField + 1) % len(m.columns)
}

// SortField returns the current sort field index.
func (m TableModel) SortField() int {
	return m.sortField
}

// Update handles keyboard navigation.
func (m TableModel) Update(msg tea.Msg) (TableModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor--
			m.clampCursor()
		case "down", "j":
			m.cursor++
			m.clampCursor()
		case "g", "home":
			m.cursor = 0
			m.offset = 0
		case "G", "end":
			m.cursor = len(m.filtered) - 1
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "pgup":
			visible := m.visibleRowCount()
			m.cursor -= visible
			m.clampCursor()
		case "pgdown":
			visible := m.visibleRowCount()
			m.cursor += visible
			m.clampCursor()
		}
	}
	return m, nil
}

// View renders the table as a string.
func (m TableModel) View() string {
	if m.width == 0 {
		return ""
	}

	var lines []string

	// Header row
	headerLine := m.renderRow(m.headerStrings(), TableHeaderStyle, false)
	lines = append(lines, headerLine)

	// Data rows
	start, end := m.visibleRange()
	for i := start; i < end; i++ {
		row := m.filtered[i]
		selected := i == m.cursor
		var style lipgloss.Style
		if selected {
			style = SelectedRowStyle
		}
		lines = append(lines, m.renderRow(row.Columns(), style, selected))
	}

	return strings.Join(lines, "\n")
}

func (m TableModel) headerStrings() []string {
	headers := make([]string, len(m.columns))
	for i, col := range m.columns {
		title := col.Title
		if i == m.sortField && title != "" {
			title += " " + DownArrow
		}
		headers[i] = title
	}
	return headers
}

func (m TableModel) renderRow(cols []string, baseStyle lipgloss.Style, selected bool) string {
	if len(m.colWidths) == 0 {
		return ""
	}

	parts := make([]string, len(m.columns))
	for i := range m.columns {
		w := 0
		if i < len(m.colWidths) {
			w = m.colWidths[i]
		}
		if w <= 0 {
			w = 1
		}

		text := ""
		if i < len(cols) {
			text = cols[i]
		}

		style := lipgloss.NewStyle().Width(w).MaxWidth(w)
		if selected {
			style = style.Inherit(SelectedRowStyle)
		} else if baseStyle.GetForeground() != nil {
			style = style.Inherit(baseStyle)
		}
		parts[i] = style.Render(text)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m *TableModel) calculateColumnWidths() {
	if m.width == 0 || len(m.columns) == 0 {
		return
	}

	m.colWidths = make([]int, len(m.columns))
	spacing := DefaultColumnSpacing * len(m.columns)
	remaining := m.width - spacing
	proportionalCount := 0

	for i, col := range m.columns {
		if col.Fixed && col.Width > 0 {
			m.colWidths[i] = col.Width
			remaining -= col.Width
		} else {
			proportionalCount++
		}
	}

	if proportionalCount > 0 && remaining > 0 {
		propWidth := remaining / proportionalCount
		for i, col := range m.columns {
			if !col.Fixed || col.Width == 0 {
				m.colWidths[i] = propWidth
			}
		}
	}
}

func (m *TableModel) applyFilter() {
	if m.filterText == "" {
		m.filtered = m.rows
		return
	}
	m.filtered = nil
	for _, row := range m.rows {
		if m.filterFn(row, m.filterText) {
			m.filtered = append(m.filtered, row)
		}
	}
}

func (m *TableModel) clampCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if len(m.filtered) > 0 && m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}

	// Adjust scroll offset to keep cursor visible
	visible := m.visibleRowCount()
	if visible <= 0 {
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

func (m TableModel) visibleRowCount() int {
	// height minus header row
	h := m.height - 1
	if h < 1 {
		h = 1
	}
	return h
}

func (m TableModel) visibleRange() (int, int) {
	count := len(m.filtered)
	if count == 0 {
		return 0, 0
	}

	visible := m.visibleRowCount()
	start := m.offset
	if start < 0 {
		start = 0
	}
	end := start + visible
	if end > count {
		end = count
	}
	return start, end
}
