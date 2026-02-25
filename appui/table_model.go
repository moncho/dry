package appui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
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

// StyledRow is an optional interface that TableRow implementations can
// use to provide a base foreground style for the row.
type StyledRow interface {
	Style() lipgloss.Style
}

// StyledIndicator is an optional interface for rows that have a specially
// styled first column (e.g. the ■ status indicator in the container list).
type StyledIndicator interface {
	IndicatorStyle() lipgloss.Style
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
	headerLine := m.renderRow(m.headerStrings(), TableHeaderStyle, false, nil)
	lines = append(lines, headerLine)

	// Data rows
	start, end := m.visibleRange()
	for i := start; i < end; i++ {
		row := m.filtered[i]
		selected := i == m.cursor
		var style lipgloss.Style
		if selected {
			style = SelectedRowStyle
		} else if sr, ok := row.(StyledRow); ok {
			style = sr.Style()
		}
		// Check for indicator styling on column 0
		var indStyle *lipgloss.Style
		if si, ok := row.(StyledIndicator); ok {
			s := si.IndicatorStyle()
			indStyle = &s
		}
		lines = append(lines, m.renderRow(row.Columns(), style, selected, indStyle))
	}

	// Pad with empty lines to fill allocated height, ensuring the footer
	// stays at the bottom of the screen.
	totalRows := 1 + (end - start) // header + data rows
	for totalRows < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
		totalRows++
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

// padOrTruncate ensures text is exactly targetWidth visual characters.
func padOrTruncate(text string, targetWidth int) string {
	if targetWidth <= 0 {
		return ""
	}
	w := ansi.StringWidth(text)
	if w > targetWidth {
		return ansi.Truncate(text, targetWidth, "")
	}
	if w < targetWidth {
		return text + strings.Repeat(" ", targetWidth-w)
	}
	return text
}

func (m TableModel) renderRow(cols []string, baseStyle lipgloss.Style, selected bool, indicatorStyle *lipgloss.Style) string {
	if len(m.colWidths) == 0 {
		return ""
	}

	// Build each column's plain text, truncated/padded to exact width.
	cells := make([]string, len(m.columns))
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

		// Truncate/pad text to column width, reserving space for column gap.
		textWidth := w - DefaultColumnSpacing
		if textWidth < 0 {
			textWidth = 0
		}
		cells[i] = padOrTruncate(text, textWidth) + strings.Repeat(" ", w-textWidth)
	}

	// For rows with an indicator column (e.g. container ■), we must style
	// each cell individually. A whole-line Render would nest ANSI resets
	// and kill the row color after the indicator's reset sequence.
	hasIndicator := !selected && indicatorStyle != nil

	if hasIndicator {
		styled := make([]string, len(cells))
		for i, cell := range cells {
			if i == 0 {
				styled[i] = indicatorStyle.Render(cell)
			} else {
				styled[i] = baseStyle.Render(cell)
			}
		}
		line := strings.Join(styled, "")
		// Pad trailing space with the row style
		lineWidth := ansi.StringWidth(line)
		if lineWidth < m.width {
			line += baseStyle.Render(strings.Repeat(" ", m.width-lineWidth))
		}
		return line
	}

	// Non-indicator rows: join as plain text, then apply one style to the whole line.
	line := strings.Join(cells, "")
	lineWidth := ansi.StringWidth(line)
	if lineWidth < m.width {
		line += strings.Repeat(" ", m.width-lineWidth)
	}

	if selected {
		return SelectedRowStyle.Render(line)
	}
	if baseStyle.GetForeground() != nil {
		return baseStyle.Render(line)
	}
	return line
}

func (m *TableModel) calculateColumnWidths() {
	if m.width == 0 || len(m.columns) == 0 {
		return
	}

	m.colWidths = make([]int, len(m.columns))
	remaining := m.width
	proportionalCount := 0
	lastProportional := -1

	for i, col := range m.columns {
		if col.Fixed && col.Width > 0 {
			w := col.Width + DefaultColumnSpacing
			m.colWidths[i] = w
			remaining -= w
		} else {
			proportionalCount++
			lastProportional = i
		}
	}

	if proportionalCount > 0 && remaining > 0 {
		propWidth := remaining / proportionalCount
		assigned := 0
		for i, col := range m.columns {
			if !col.Fixed || col.Width == 0 {
				if i == lastProportional {
					// Give the last proportional column the remainder
					// to avoid rounding gaps.
					m.colWidths[i] = remaining - assigned
				} else {
					m.colWidths[i] = propWidth
					assigned += propWidth
				}
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
