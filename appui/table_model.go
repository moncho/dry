package appui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
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
// It wraps bubbles/table for rendering and keyboard navigation while keeping
// sort, filter, and column-width logic locally.
type TableModel struct {
	inner      table.Model
	columns    []Column
	rows       []TableRow
	filtered   []TableRow
	sortField  int
	sortAsc    bool
	filterText string
	filterFn   func(row TableRow, pattern string) bool
	colWidths  []int
	width      int
	height     int
}

// NewTableModel creates a table with the given column definitions.
func NewTableModel(columns []Column) TableModel {
	t := table.New(table.WithFocused(true))

	km := table.DefaultKeyMap()
	km.LineUp = key.NewBinding(key.WithKeys("up", "k"))
	km.LineDown = key.NewBinding(key.WithKeys("down", "j"))
	km.PageUp = key.NewBinding(key.WithKeys("pgup"))
	km.PageDown = key.NewBinding(key.WithKeys("pgdown"))
	km.GotoTop = key.NewBinding(key.WithKeys("g", "home"))
	km.GotoBottom = key.NewBinding(key.WithKeys("G", "end"))
	km.HalfPageUp = key.NewBinding(key.WithKeys())  // disable
	km.HalfPageDown = key.NewBinding(key.WithKeys()) // disable
	t.KeyMap = km

	t.SetStyles(table.Styles{
		Header:   TableHeaderStyle,
		Cell:     lipgloss.NewStyle(),
		Selected: SelectedRowStyle,
	})

	return TableModel{
		inner:    t,
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
	m.syncInner()
}

// SetSize updates the table dimensions.
func (m *TableModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.calculateColumnWidths()
	m.inner.SetWidth(w)
	m.inner.SetHeight(h)
	m.syncInnerColumns()
}

// SelectedRow returns the row under the cursor, or nil.
func (m TableModel) SelectedRow() TableRow {
	cursor := m.inner.Cursor()
	if cursor >= 0 && cursor < len(m.filtered) {
		return m.filtered[cursor]
	}
	return nil
}

// Cursor returns the current cursor position.
func (m TableModel) Cursor() int {
	return m.inner.Cursor()
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
	m.syncInner()
}

// NextSort cycles to the next sort field and re-sorts the rows.
func (m *TableModel) NextSort() {
	m.sortField = (m.sortField + 1) % len(m.columns)
	m.sortRows()
	m.syncInnerColumns()
}

// SortField returns the current sort field index.
func (m TableModel) SortField() int {
	return m.sortField
}

// Update handles keyboard navigation via the inner bubbles table.
func (m TableModel) Update(msg tea.Msg) (TableModel, tea.Cmd) {
	var cmd tea.Cmd
	m.inner, cmd = m.inner.Update(msg)
	return m, cmd
}

// View renders the table as a string.
func (m TableModel) View() string {
	if m.width == 0 {
		return ""
	}

	view := m.inner.View()

	// Pad each line to the full terminal width so backgrounds extend.
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		w := ansi.StringWidth(line)
		if w < m.width {
			lines[i] = line + strings.Repeat(" ", m.width-w)
		}
	}

	// Pad with empty lines to fill allocated height so the footer stays
	// at the bottom of the screen.
	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", m.width))
	}

	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (m *TableModel) syncInner() {
	m.inner.SetRows(m.toBubblesRows())
	if cursor := m.inner.Cursor(); cursor >= len(m.filtered) && len(m.filtered) > 0 {
		m.inner.SetCursor(len(m.filtered) - 1)
	}
}

func (m *TableModel) syncInnerColumns() {
	cols := make([]table.Column, len(m.columns))
	totalW := 0
	for i, c := range m.columns {
		w := 0
		if i < len(m.colWidths) {
			w = m.colWidths[i]
		}
		title := c.Title
		if i == m.sortField && title != "" {
			title += " " + DownArrow
		}
		cols[i] = table.Column{Title: title, Width: w}
		totalW += w
	}
	// Stretch the last column so the total equals m.width, ensuring the
	// selected-row highlight extends to the full screen width.
	if totalW < m.width && len(cols) > 0 {
		cols[len(cols)-1].Width += m.width - totalW
	}
	m.inner.SetColumns(cols)
}

func (m *TableModel) toBubblesRows() []table.Row {
	rows := make([]table.Row, len(m.filtered))
	for i, r := range m.filtered {
		cols := r.Columns()
		row := make(table.Row, len(m.columns))
		var rowStyle lipgloss.Style
		if sr, ok := r.(StyledRow); ok {
			rowStyle = sr.Style()
		}
		for j := range m.columns {
			cell := ""
			if j < len(cols) {
				cell = cols[j]
			}
			if j == 0 {
				if si, ok := r.(StyledIndicator); ok {
					row[j] = si.IndicatorStyle().Render(cell)
					continue
				}
			}
			if rowStyle.GetForeground() != nil {
				row[j] = rowStyle.Render(cell)
			} else {
				row[j] = cell
			}
		}
		rows[i] = row
	}
	return rows
}

func (m *TableModel) sortRows() {
	col := m.sortField
	asc := m.sortAsc
	sort.SliceStable(m.rows, func(i, j int) bool {
		ci := colValue(m.rows[i], col)
		cj := colValue(m.rows[j], col)
		if asc {
			return ci < cj
		}
		return ci > cj
	})
	m.applyFilter()
	m.syncInner()
}

func colValue(row TableRow, col int) string {
	cols := row.Columns()
	if col < len(cols) {
		return strings.ToLower(cols[col])
	}
	return ""
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
