package appui

import (
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Column defines a table column.
type Column struct {
	Title string
	Width int  // 0 means proportional (share remaining space)
	Fixed bool // fixed-width column
}

// minProportionalColumnWidth is the smallest allocation a proportional
// column may get when fixed columns leave no remaining space. It is nine
// visible cells plus the spacing gutter, except for the last column, which
// fitCell does not fit at all: there it is ten visible cells and no gutter,
// which costs nothing, since nothing sits to its right.
const minProportionalColumnWidth = 10

// TableRow represents one row of data in the table.
type TableRow interface {
	Columns() []string
	ID() string
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
	// contentWidths is how much of each allocation the text may use: the
	// allocation less the gutter, zero (no limit) for the last column.
	contentWidths []int
	width         int
	height        int
}

// NewTableModel creates a table with the given column definitions.
func NewTableModel(columns []Column) TableModel {
	t := table.New(table.WithFocused(true))

	km := table.DefaultKeyMap()
	km.LineUp = key.NewBinding(key.WithKeys("up", "k"))
	km.LineDown = key.NewBinding(key.WithKeys("down", "j"))
	// pgup alone, without bubbles' "b", and HalfPageUp/Down disabled
	// below: appui/compose's movesUp enumerates the upward keys to decide
	// which way to skip a section header, so this binding is load-bearing
	// outside this package.
	km.PageUp = key.NewBinding(key.WithKeys("pgup"))
	km.PageDown = key.NewBinding(key.WithKeys("pgdown"))
	km.GotoTop = key.NewBinding(key.WithKeys("g", "home"))
	km.GotoBottom = key.NewBinding(key.WithKeys("G", "end"))
	km.HalfPageUp = key.NewBinding(key.WithKeys())   // disable
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

// SetSize updates the table dimensions. Table height is reduced
// by 1 to leave space for the blank line after the table.
func (m *TableModel) SetSize(w, h int) {
	widthChanged := m.width != w
	m.width = w
	m.height = h
	m.calculateColumnWidths()
	m.syncInnerColumns()
	// Cells are fitted to the column widths, so a new width refits them:
	// rows set at the old one would stay truncated after a resize.
	if widthChanged {
		m.syncInner()
	}
	m.inner.SetWidth(w)
	// -1 for blank line after the table
	m.inner.SetHeight(h - 1)
	m.ensureCursorVisible()
}

// ColumnWidth is the width allocated to a column, which is where its
// boundary falls, except for the last column: syncInnerColumns renders that
// one wider when the columns fit with room to spare. Exported for tests
// only, in appui/compose, appui/swarm and the monitor's own; no production
// code calls it.
func (m TableModel) ColumnWidth(i int) int {
	if i < 0 || i >= len(m.colWidths) {
		return 0
	}
	return m.colWidths[i]
}

// Width returns the table's current width.
func (m TableModel) Width() int { return m.width }

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

// SetCursor moves the cursor to the given row index, clamped to the visible
// rows, keeping the row on screen. It moves by a delta because MoveUp and
// MoveDown compensate the scroll offset and the inner table's own SetCursor
// does not, which would highlight a row off screen while every key kept
// acting on it. A delta of any size is enough, wider than the window
// included, which TestTableModel_ADeltaMoveKeepsTheCursorOnScreen asserts.
func (m *TableModel) SetCursor(i int) {
	switch delta := i - m.inner.Cursor(); {
	case delta > 0:
		m.inner.MoveDown(delta)
	case delta < 0:
		m.inner.MoveUp(-delta)
	}
}

// ensureCursorVisible scrolls the viewport so the cursor's row is rendered.
// A viewport that shrinks under the cursor on a resize leaves the selection
// off screen, and re-walking from the top is the only way to make the inner
// table recompute its offset. That walk ends on the last visible line, so
// the check has to come first: a resize that leaves the selection on screen
// must not scroll.
func (m *TableModel) ensureCursorVisible() {
	cursor := m.inner.Cursor()
	if cursor <= 0 || m.cursorOnScreen() {
		return
	}
	m.inner.GotoTop()
	m.inner.MoveDown(cursor)
}

// cursorOnScreen reports whether the cursor's row is among the ones the
// inner table renders. The offset is not exported, so the answer is read
// back out of the render, by the escape sequence the Selected style emits.
// A theme that renders nothing distinguishable answers yes, which is the
// behaviour of not correcting at all.
func (m TableModel) cursorOnScreen() bool {
	// A list that fits needs no render: every row is on screen. Capacity is
	// height-2, not height-1: SetSize gives the inner table h-1, and bubbles
	// takes another line off for the header.
	if len(m.filtered) <= max(m.height-2, 0) {
		return true
	}
	marker := selectedRowMarker()
	if marker == "" {
		return true
	}
	return selectedRowVisible(strings.Split(m.inner.View(), "\n"), marker)
}

// selectedRowVisible reports whether any line begins with the marker the
// inner table emits before the cursor's row. A prefix rather than a search:
// cell content arrives pre-styled from several views, and a cell carrying
// the same sequence must not answer for the cursor.
func selectedRowVisible(lines []string, marker string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, marker) {
			return true
		}
	}
	return false
}

// selectedRowMarker is the escape sequence the inner table emits before the
// cursor's row, or "" when the Selected style renders nothing.
func selectedRowMarker() string {
	const probe = "\x00"
	rendered := SelectedRowStyle.Render(probe)
	if i := strings.Index(rendered, probe); i > 0 {
		return rendered[:i]
	}
	return ""
}

// SelectRowByID moves the cursor onto the visible row with the given ID and
// reports whether it was found, so a view rebuilt on every refresh can
// follow the row rather than the index it happened to occupy.
func (m *TableModel) SelectRowByID(id string) bool {
	for i, row := range m.filtered {
		if row.ID() == id {
			m.SetCursor(i)
			return true
		}
	}
	return false
}

// RowCount returns the number of visible (filtered) rows.
func (m TableModel) RowCount() int {
	return len(m.filtered)
}

// TotalRowCount returns the total number of unfiltered rows.
func (m TableModel) TotalRowCount() int {
	return len(m.rows)
}

// FilteredRows returns the currently visible (filtered) rows.
func (m TableModel) FilteredRows() []TableRow {
	return m.filtered
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

// SetSortField sets the sort field to a specific column index and updates
// the column header indicator without performing a local sort. Use this when
// sorting is handled externally (e.g., server-side Docker sort).
func (m *TableModel) SetSortField(col int) {
	if col >= 0 && col < len(m.columns) {
		m.sortField = col
	} else {
		m.sortField = -1
	}
	m.syncInnerColumns()
}

// SortField returns the current sort field index.
func (m TableModel) SortField() int {
	return m.sortField
}

// Sort re-sorts the current rows using the active sort field.
func (m *TableModel) Sort() {
	m.sortRows()
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
	// The selected row (cursor+1, accounting for the header line) must
	// use SelectedRowStyle so the highlight covers the full width.
	lines := strings.Split(view, "\n")
	selectedLine := m.inner.Cursor() + 1 // +1 for header row
	for i, line := range lines {
		w := ansi.StringWidth(line)
		if w > m.width {
			// Only the header reaches here: bubbles renders it outside
			// the viewport, which clips every data row to the width
			// first. Marking it is what says a column was dropped, since
			// the rows below lose theirs silently.
			lines[i] = ansi.Truncate(line, m.width, "…")
		} else if w < m.width {
			pad := strings.Repeat(" ", m.width-w)
			if i == selectedLine {
				pad = SelectedRowStyle.Render(pad)
			}
			lines[i] = line + pad
		}
	}

	// Pad with empty lines to fill allocated height so the footer stays at
	// the bottom of the screen. max() is defensive: only a zero width
	// returns early above, and strings.Repeat panics on a negative count.
	// No layout produces a negative width today, so nothing tests it.
	for len(lines) < m.height {
		lines = append(lines, strings.Repeat(" ", max(m.width, 0)))
	}

	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (m *TableModel) syncInner() {
	// bubbles' table indexes its column slice per cell, so rows set before
	// the first SetSize would panic there. SetSize pushes them.
	if len(m.colWidths) == 0 {
		return
	}
	m.inner.SetRows(m.toBubblesRows())
	cursor := m.inner.Cursor()
	if cursor < 0 && len(m.filtered) > 0 {
		m.inner.SetCursor(0)
	} else if cursor >= len(m.filtered) && len(m.filtered) > 0 {
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
		if m.sortField >= 0 && i == m.sortField && title != "" {
			title += " " + DownArrow
		}
		cols[i] = table.Column{Title: m.fitCell(title, i), Width: w}
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
		for j := range m.columns {
			if j < len(cols) {
				// Skip ColorFg wrapping for columns that already contain
				// ANSI escape sequences (e.g. container status indicator)
				// to avoid double-coloring. Asked of the cell as it
				// arrived, not of the fitted one.
				cell := m.fitCell(cols[j], j)
				if strings.Contains(cols[j], "\x1b[") {
					row[j] = cell
				} else {
					row[j] = ColorFg(cell, DryTheme.Fg)
				}
			}
		}
		rows[i] = row
	}
	return rows
}

// fitCell shortens a cell, or a header title, so the column keeps a spacing
// gutter on its right. bubbles' table pads every cell to its column width
// and joins them with no separator, so the gap is whatever the left cell
// leaves unused. The last column is not fitted: nothing sits to its right,
// and when the columns fit with room to spare syncInnerColumns stretches it
// past its allocation.
func (m *TableModel) fitCell(s string, col int) string {
	if col >= len(m.contentWidths) {
		return s
	}
	limit := m.contentWidths[col]
	if limit <= 0 || ansi.StringWidth(s) <= limit {
		return s
	}
	return ansi.Truncate(s, limit, "…")
}

func (m *TableModel) sortRows() {
	col := m.sortField
	asc := m.sortAsc
	sort.SliceStable(m.rows, func(i, j int) bool {
		ci := colValue(m.rows[i], col)
		cj := colValue(m.rows[j], col)
		// Try numeric comparison so "9" < "10".
		if ni, ei := strconv.ParseFloat(ci, 64); ei == nil {
			if nj, ej := strconv.ParseFloat(cj, 64); ej == nil {
				if asc {
					return ni < nj
				}
				return ni > nj
			}
		}
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
		// Strip ANSI escape sequences so sorting compares visible text only.
		return strings.ToLower(ansi.Strip(cols[col]))
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

// RefreshStyles re-applies the current theme styles to the inner table.
// Call after InitStyles() to pick up theme changes.
func (m *TableModel) RefreshStyles() {
	m.inner.SetStyles(table.Styles{
		Header:   TableHeaderStyle,
		Cell:     lipgloss.NewStyle(),
		Selected: SelectedRowStyle,
	})
	// Re-convert rows so ColorFg picks up the new theme foreground.
	m.syncInner()
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

	if proportionalCount > 0 {
		// A proportional column must never collapse to zero width: bubbles'
		// table silently drops zero-width columns, making them invisible.
		// While space remains, shrink to fit so every column stays on
		// screen; only when fixed columns alone already overflow does each
		// proportional column get the minimum, with View truncating the
		// overflow on the right.
		//
		// The gutter comes out of the content, not the allocation (see
		// fitCell), so only a one-cell column goes without one. Flooring
		// the allocation to buy that gutter would overflow the table
		// further than it already does at these widths, and each cell of
		// overflow costs the rightmost column: see
		// TestTableModel_FittingIsNeverTradedForSpacing.
		propWidth := minProportionalColumnWidth
		if remaining > 0 {
			propWidth = max(remaining/proportionalCount, 1)
		}
		assigned := 0
		for i, col := range m.columns {
			if !col.Fixed || col.Width == 0 {
				if i == lastProportional && remaining-assigned > propWidth {
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

	// The allocation less the gutter, and zero (no limit) for the last
	// column, which syncInnerColumns stretches to fill the table.
	m.contentWidths = make([]int, len(m.colWidths))
	last := len(m.colWidths) - 1
	for i, w := range m.colWidths {
		if i == last {
			continue
		}
		m.contentWidths[i] = max(w-DefaultColumnSpacing, 1)
	}
}
