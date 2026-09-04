package appui

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type testRow struct {
	id   string
	cols []string
}

func (r testRow) Columns() []string { return r.cols }
func (r testRow) ID() string        { return r.id }

func makeRows(n int) []TableRow {
	rows := make([]TableRow, n)
	for i := range n {
		rows[i] = testRow{
			id:   string(rune('a' + i)),
			cols: []string{string(rune('A' + i)), "col2"},
		}
	}
	return rows
}

func TestTableModel_CursorNavigation(t *testing.T) {
	cols := []Column{{Title: "Name"}, {Title: "Value"}}
	table := NewTableModel(cols)
	table.SetSize(80, 25)
	table.SetRows(makeRows(5))

	// Initial cursor at 0
	if table.Cursor() != 0 {
		t.Fatalf("expected cursor 0, got %d", table.Cursor())
	}

	// Move down
	table, _ = table.Update(tea.KeyPressMsg{Code: 'j'})
	if table.Cursor() != 1 {
		t.Fatalf("expected cursor 1 after j, got %d", table.Cursor())
	}

	// Move up with arrow
	table, _ = table.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if table.Cursor() != 0 {
		t.Fatalf("expected cursor 0 after up, got %d", table.Cursor())
	}

	// Move to end
	table, _ = table.Update(tea.KeyPressMsg{Code: 'G'})
	if table.Cursor() != 4 {
		t.Fatalf("expected cursor 4 after G, got %d", table.Cursor())
	}

	// Move to beginning
	table, _ = table.Update(tea.KeyPressMsg{Code: 'g'})
	if table.Cursor() != 0 {
		t.Fatalf("expected cursor 0 after g, got %d", table.Cursor())
	}
}

func TestTableModel_CursorClamp(t *testing.T) {
	cols := []Column{{Title: "Name"}}
	table := NewTableModel(cols)
	table.SetSize(80, 25)
	table.SetRows(makeRows(3))

	// Can't go below 0
	table, _ = table.Update(tea.KeyPressMsg{Code: 'k'})
	if table.Cursor() != 0 {
		t.Fatalf("cursor should clamp to 0, got %d", table.Cursor())
	}

	// Can't go past end
	table, _ = table.Update(tea.KeyPressMsg{Code: 'G'})
	table, _ = table.Update(tea.KeyPressMsg{Code: 'j'})
	if table.Cursor() != 2 {
		t.Fatalf("cursor should clamp to 2, got %d", table.Cursor())
	}
}

func TestTableModel_EmptyTable(t *testing.T) {
	cols := []Column{{Title: "Name"}}
	table := NewTableModel(cols)
	table.SetSize(80, 25)
	table.SetRows(nil)

	if table.RowCount() != 0 {
		t.Fatalf("expected 0 rows, got %d", table.RowCount())
	}
	if table.SelectedRow() != nil {
		t.Fatal("expected nil selected row for empty table")
	}
}

func TestTableModel_SelectedRow(t *testing.T) {
	cols := []Column{{Title: "Name"}}
	table := NewTableModel(cols)
	table.SetSize(80, 25)
	table.SetRows(makeRows(3))

	row := table.SelectedRow()
	if row == nil {
		t.Fatal("expected non-nil selected row")
	}
	if row.ID() != "a" {
		t.Fatalf("expected first row ID 'a', got %q", row.ID())
	}

	table, _ = table.Update(tea.KeyPressMsg{Code: 'j'})
	row = table.SelectedRow()
	if row.ID() != "b" {
		t.Fatalf("expected second row ID 'b', got %q", row.ID())
	}
}

func TestTableModel_Filter(t *testing.T) {
	cols := []Column{{Title: "Name"}, {Title: "Value"}}
	table := NewTableModel(cols)
	table.SetSize(80, 25)
	rows := []TableRow{
		testRow{id: "1", cols: []string{"alpha", "x"}},
		testRow{id: "2", cols: []string{"beta", "y"}},
		testRow{id: "3", cols: []string{"gamma", "x"}},
	}
	table.SetRows(rows)

	if table.RowCount() != 3 {
		t.Fatalf("expected 3 rows, got %d", table.RowCount())
	}

	// Filter to rows containing "ph" — only "alpha"
	table.SetFilter("ph")
	if table.RowCount() != 1 {
		t.Fatalf("expected 1 filtered row (alpha), got %d", table.RowCount())
	}
	if table.TotalRowCount() != 3 {
		t.Fatalf("total should still be 3, got %d", table.TotalRowCount())
	}

	// Clear filter
	table.SetFilter("")
	if table.RowCount() != 3 {
		t.Fatalf("expected 3 rows after clearing filter, got %d", table.RowCount())
	}
}

func TestTableModel_NextSort(t *testing.T) {
	cols := []Column{{Title: "A"}, {Title: "B"}, {Title: "C"}}
	table := NewTableModel(cols)

	if table.SortField() != 0 {
		t.Fatalf("expected initial sort field 0, got %d", table.SortField())
	}

	table.NextSort()
	if table.SortField() != 1 {
		t.Fatalf("expected sort field 1, got %d", table.SortField())
	}

	table.NextSort()
	if table.SortField() != 2 {
		t.Fatalf("expected sort field 2, got %d", table.SortField())
	}

	table.NextSort()
	if table.SortField() != 0 {
		t.Fatalf("expected sort field 0 after wrap, got %d", table.SortField())
	}
}

func TestTableModel_SetSortField(t *testing.T) {
	cols := []Column{{Title: "A"}, {Title: "B"}, {Title: "C"}, {Title: "D"}}
	table := NewTableModel(cols)
	table.SetSize(80, 25)
	table.SetRows(makeRows(3))

	// Set sort field to column 2
	table.SetSortField(2)
	if table.SortField() != 2 {
		t.Fatalf("expected sort field 2, got %d", table.SortField())
	}

	// Set sort field to -1 (no sort indicator)
	table.SetSortField(-1)
	if table.SortField() != -1 {
		t.Fatalf("expected sort field -1, got %d", table.SortField())
	}

	// Out-of-range sets to -1
	table.SetSortField(99)
	if table.SortField() != -1 {
		t.Fatalf("expected sort field -1 for out-of-range, got %d", table.SortField())
	}

	// Valid field works again
	table.SetSortField(0)
	if table.SortField() != 0 {
		t.Fatalf("expected sort field 0, got %d", table.SortField())
	}
}

func TestTableModel_ScrollOffset(t *testing.T) {
	cols := []Column{{Title: "Name"}}
	table := NewTableModel(cols)
	// Height 5 = 4 visible rows (minus header)
	table.SetSize(80, 5)
	table.SetRows(makeRows(10))

	// Move to row 5 (past visible window)
	for range 5 {
		table, _ = table.Update(tea.KeyPressMsg{Code: 'j'})
	}
	if table.Cursor() != 5 {
		t.Fatalf("expected cursor 5, got %d", table.Cursor())
	}

	// The inner bubbles table guarantees the cursor row is visible.
	// Verify the view is non-empty as a sanity check.
	if table.View() == "" {
		t.Fatal("View() should not be empty after scrolling")
	}
}

func TestTableModel_Sort(t *testing.T) {
	cols := []Column{{Title: "Name"}, {Title: "Value"}}
	table := NewTableModel(cols)
	table.SetSize(80, 25)
	rows := []TableRow{
		testRow{id: "1", cols: []string{"cherry", "3"}},
		testRow{id: "2", cols: []string{"apple", "1"}},
		testRow{id: "3", cols: []string{"banana", "2"}},
	}
	table.SetRows(rows)

	// Sort by first column (Name)
	table.Sort()
	if table.RowCount() != 3 {
		t.Fatalf("expected 3 rows, got %d", table.RowCount())
	}
	// Default sort field is 0 (Name), ascending
	r := table.SelectedRow()
	if r == nil || r.Columns()[0] != "apple" {
		t.Fatalf("expected first row 'apple' after sort, got %v", r)
	}
}

func TestTableModel_SortNumeric(t *testing.T) {
	cols := []Column{{Title: "Name"}, {Title: "Count"}}
	table := NewTableModel(cols)
	table.SetSize(80, 25)
	rows := []TableRow{
		testRow{id: "1", cols: []string{"a", "9"}},
		testRow{id: "2", cols: []string{"b", "10"}},
		testRow{id: "3", cols: []string{"c", "2"}},
		testRow{id: "4", cols: []string{"d", "100"}},
	}
	table.SetRows(rows)

	// Sort by column 1 (Count) — should use numeric comparison
	table.NextSort() // sort field = 1
	table.Sort()

	// Numeric ascending: 2, 9, 10, 100
	expected := []string{"2", "9", "10", "100"}
	for i, want := range expected {
		r := table.filtered[i]
		got := r.Columns()[1]
		if got != want {
			t.Errorf("row %d: expected count %q, got %q", i, want, got)
		}
	}
}

func TestTableModel_SortNumericWithANSI(t *testing.T) {
	cols := []Column{{Title: "Name"}, {Title: "Pids"}}
	table := NewTableModel(cols)
	table.SetSize(80, 25)
	// Wrap numbers in ANSI color codes like monitor model does
	rows := []TableRow{
		testRow{id: "1", cols: []string{"a", ColorFg("9", DryTheme.FgMuted)}},
		testRow{id: "2", cols: []string{"b", ColorFg("10", DryTheme.FgMuted)}},
		testRow{id: "3", cols: []string{"c", ColorFg("2", DryTheme.FgMuted)}},
		testRow{id: "4", cols: []string{"d", ColorFg("100", DryTheme.FgMuted)}},
	}
	table.SetRows(rows)

	// Sort by column 1 (Pids)
	table.NextSort() // sort field = 1
	table.Sort()

	// Numeric ascending: 2, 9, 10, 100
	expectedIDs := []string{"3", "1", "2", "4"}
	for i, want := range expectedIDs {
		r := table.filtered[i]
		if r.ID() != want {
			t.Errorf("row %d: expected id %q, got %q", i, want, r.ID())
		}
	}
}

func TestTableModel_SortPreservedAfterSetRows(t *testing.T) {
	cols := []Column{{Title: "Name"}, {Title: "Value"}}
	table := NewTableModel(cols)
	table.SetSize(80, 25)

	// Set sort field to column 1 (Value)
	table.NextSort() // now sorting by column 1

	rows := []TableRow{
		testRow{id: "1", cols: []string{"cherry", "c"}},
		testRow{id: "2", cols: []string{"apple", "a"}},
		testRow{id: "3", cols: []string{"banana", "b"}},
	}
	table.SetRows(rows)
	// SetRows does NOT re-sort, so order is insertion order
	r := table.SelectedRow()
	if r == nil || r.ID() != "1" {
		t.Fatalf("expected first row id '1' after SetRows (no sort), got %v", r)
	}

	// Calling Sort() should re-sort by column 1 (Value)
	table.Sort()
	r = table.SelectedRow()
	if r == nil || r.Columns()[1] != "a" {
		t.Fatalf("expected first row value 'a' after Sort(), got %v", r)
	}
}

func TestTableModel_ProportionalColumnNeverCollapses(t *testing.T) {
	// When fixed columns consume the full width, a proportional column must
	// keep a minimum width — bubbles' table silently drops zero-width
	// columns from the rendered view.
	cols := []Column{
		{Title: "A", Width: 60, Fixed: true},
		{Title: "NAME"},
		{Title: "B", Width: 60, Fixed: true},
	}
	table := NewTableModel(cols)
	table.SetSize(100, 10)
	// 10 as a literal, for the same reason: read from the constant, this
	// assertion moves whenever the constant does.
	if got := table.colWidths[1]; got != 10 {
		t.Fatalf("expected the 10-cell minimum when fixed columns overflow, got %d", got)
	}

	// With room to spare, the proportional column takes the remainder.
	table.SetSize(200, 10)
	want := 200 - (60 + DefaultColumnSpacing) - (60 + DefaultColumnSpacing)
	if got := table.colWidths[1]; got != want {
		t.Fatalf("expected proportional column width %d at width 200, got %d", want, got)
	}
}

func TestTableModel_ProportionalColumnsShrinkToFitWhenSpaceRemains(t *testing.T) {
	// When fixed columns leave a little space, proportional columns shrink
	// to fit instead of being floored to the minimum: flooring would push
	// trailing columns (and their headers) off screen with no indication.
	cols := []Column{
		{Title: "A", Width: 40, Fixed: true},
		{Title: "NAME"},
		{Title: "SUBNET"},
	}
	table := NewTableModel(cols)
	table.SetSize(50, 10) // fixed consumes 41, leaving 9 for two columns

	total := 0
	for i, w := range table.colWidths {
		if i > 0 && w < 1 {
			t.Fatalf("proportional column %d collapsed to width %d", i, w)
		}
		total += w
	}
	if total > 50 {
		t.Fatalf("columns overflow the table width despite available space: total %d > 50", total)
	}
}

func TestTableModel_ViewTruncatesOverflowLines(t *testing.T) {
	cols := []Column{
		{Title: "A", Width: 60, Fixed: true},
		{Title: "NAME"},
		{Title: "B", Width: 60, Fixed: true},
	}
	table := NewTableModel(cols)
	table.SetSize(100, 10)
	table.SetRows([]TableRow{testRow{id: "1", cols: []string{
		strings.Repeat("x", 60), "name-1", strings.Repeat("y", 60),
	}}})

	for i, line := range strings.Split(table.View(), "\n") {
		if got := ansi.StringWidth(line); got > 100 {
			t.Fatalf("line %d: width %d exceeds table width 100", i, got)
		}
	}
}

func TestTableModel_ViewNotEmpty(t *testing.T) {
	cols := []Column{{Title: "Name", Width: 20, Fixed: true}}
	table := NewTableModel(cols)
	table.SetSize(80, 10)
	table.SetRows(makeRows(3))

	view := table.View()
	if view == "" {
		t.Fatal("View() should not be empty")
	}
}

// selectedStylePrefix is the escape sequence the inner table emits for the
// row under the cursor, used to find that row in rendered output.
func selectedStylePrefix() string {
	return strings.SplitN(SelectedRowStyle.Render("x"), "x", 2)[0]
}

// SetCursor has to scroll, not just point. bubbles' own SetCursor leaves the
// viewport's offset alone, so a jump past the last visible row highlights a
// row that is not rendered: the view shows no selection at all while every
// key acts on an invisible row.
func TestTableModel_SetCursorScrollsTheRowIntoView(t *testing.T) {
	cols := []Column{{Title: "Name"}, {Title: "Value"}}
	table := NewTableModel(cols)
	table.SetSize(80, 10)
	rows := make([]TableRow, 40)
	for i := range rows {
		rows[i] = testRow{id: fmt.Sprint(i), cols: []string{fmt.Sprintf("row-%02d", i), "v"}}
	}
	table.SetRows(rows)

	table.SetCursor(37)
	if got := table.Cursor(); got != 37 {
		t.Fatalf("expected the cursor at 37, got %d", got)
	}

	view := table.View()
	if !strings.Contains(ansi.Strip(view), "row-37") {
		t.Fatalf("expected the selected row to be rendered, got:\n%s", ansi.Strip(view))
	}
	var selected string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, selectedStylePrefix()) {
			selected = ansi.Strip(line)
			break
		}
	}
	if !strings.Contains(selected, "row-37") {
		t.Fatalf("expected the highlight on row-37, got %q", selected)
	}

	// And back up again, to the top of the list.
	table.SetCursor(0)
	if !strings.Contains(ansi.Strip(table.View()), "row-00") {
		t.Fatalf("expected the first row to be rendered after scrolling back, got:\n%s",
			ansi.Strip(table.View()))
	}
}

// A viewport that shrinks under a cursor near the bottom of a list that
// used to fit leaves the selection off screen: bubbles' table adjusts its
// scroll offset only inside MoveUp and MoveDown, and SetHeight touches
// neither. The result is worse than an inert selection, a highlighted row
// nobody can see, while every key still acts on it.
func TestTableModel_ShrinkingKeepsTheSelectionOnScreen(t *testing.T) {
	InitStyles()
	table := NewTableModel([]Column{{Title: "Name"}})
	table.SetSize(80, 40)
	rows := make([]TableRow, 25)
	for i := range rows {
		rows[i] = testRow{id: fmt.Sprint(i), cols: []string{fmt.Sprintf("row-%02d", i)}}
	}
	table.SetRows(rows)
	table.SetCursor(len(rows) - 1)

	for _, height := range []int{40, 30, 20, 16, 12, 8, 4} {
		table.SetSize(80, height)
		view := table.View()
		if !strings.Contains(ansi.Strip(view), "row-24") {
			t.Fatalf("height %d: the selected row is not rendered:\n%s", height, ansi.Strip(view))
		}
		var highlighted string
		for _, line := range strings.Split(view, "\n") {
			if strings.Contains(line, selectedStylePrefix()) {
				highlighted = ansi.Strip(line)
				break
			}
		}
		if !strings.Contains(highlighted, "row-24") {
			t.Fatalf("height %d: expected the highlight on row-24, got %q", height, highlighted)
		}
	}
}

// A resize that leaves the selection on screen must leave the scroll
// position alone. The recovery walk that pulls an off-screen cursor back
// ends with it on the last visible line, so running it unconditionally
// would scroll every list on every SetSize, and SetSize runs for all
// twelve views on every window-size message, on f7, and whenever the
// workspace layout changes. The list lurching under a user who only
// resized their terminal is worse than the bug the walk exists for.
func TestTableModel_ResizeThatChangesNothingDoesNotScroll(t *testing.T) {
	InitStyles()
	table := NewTableModel([]Column{{Title: "Name"}})
	table.SetSize(80, 20)
	rows := make([]TableRow, 60)
	for i := range rows {
		rows[i] = testRow{id: fmt.Sprint(i), cols: []string{fmt.Sprintf("row-%02d", i)}}
	}
	table.SetRows(rows)

	// To the bottom, then back up into the middle of the window.
	table, _ = table.Update(tea.KeyPressMsg{Code: 'G'})
	for range 6 {
		table, _ = table.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	}
	before := ansi.Strip(table.View())

	table.SetSize(80, 20) // the same size, twice over
	table.SetSize(80, 20)
	if after := ansi.Strip(table.View()); after != before {
		t.Fatalf("a no-op resize scrolled the list:\nbefore:\n%s\nafter:\n%s", before, after)
	}

	// Moving the cursor within the window does not scroll it either: the
	// rows on screen stay where they are while the cursor travels.
	cursor := table.Cursor()
	table.SetCursor(cursor - 1)
	afterCursor := strings.Split(ansi.Strip(table.View()), "\n")[1]
	if want := strings.Split(before, "\n")[1]; afterCursor != want {
		t.Fatalf("moving the cursor to a visible row scrolled the list: %q, want %q", afterCursor, want)
	}

	// And a resize that really does hide the selection still recovers it.
	table, _ = table.Update(tea.KeyPressMsg{Code: 'G'})
	table.SetSize(80, 8)
	if !strings.Contains(ansi.Strip(table.View()), "row-59") {
		t.Fatalf("expected the selection back on screen after a shrink:\n%s", ansi.Strip(table.View()))
	}
}

// The row capacity of a table of height h is h-2: SetSize hands the inner
// table h-1 for the trailing blank line, and bubbles takes another line for
// the header. cursorOnScreen short-circuits on a list that fits, so an
// off-by-one there answers "on screen" for a list one row too long and
// ensureCursorVisible skips the correction. Only the exact boundary shows
// it, which is why this walks the three heights around it rather than the
// wide steps the other shrink test takes.
func TestTableModel_ShrinkingToExactlyOneRowTooManyStillCorrects(t *testing.T) {
	InitStyles()
	const rowCount = 25
	for _, height := range []int{rowCount, rowCount + 1, rowCount + 2} {
		m := NewTableModel([]Column{{Title: "NAME"}})
		m.SetSize(40, 40)
		rows := make([]TableRow, rowCount)
		for i := range rows {
			name := fmt.Sprintf("row-%02d", i)
			rows[i] = testRow{id: name, cols: []string{name}}
		}
		m.SetRows(rows)
		m.SetCursor(rowCount - 1)

		m.SetSize(40, height)

		last := fmt.Sprintf("row-%02d", rowCount-1)
		if !strings.Contains(ansi.Strip(m.View()), last) {
			t.Errorf("height %d: the selected row %s is not rendered after the shrink", height, last)
		}
		if !m.cursorOnScreen() {
			t.Errorf("height %d: cursorOnScreen disagrees with the render", height)
		}
	}
}

// SetCursor moves by a delta and does not correct afterwards, which is only
// safe because MoveUp and MoveDown compensate the scroll offset for a jump
// of any size, wider than the window included. That is a property of
// bubbles' table, not of this package, so it is asserted rather than
// assumed: if a bubbles upgrade breaks it, this fails and SetCursor needs
// its ensureCursorVisible call back.
func TestTableModel_ADeltaMoveKeepsTheCursorOnScreen(t *testing.T) {
	InitStyles()
	checked := 0
	for _, height := range []int{3, 5, 10, 25} {
		for _, rowCount := range []int{1, 5, 25, 60} {
			for _, filter := range []string{"", "row-0"} {
				m := NewTableModel([]Column{{Title: "NAME"}})
				m.SetSize(40, height)
				rows := make([]TableRow, rowCount)
				for i := range rows {
					name := fmt.Sprintf("row-%03d", i)
					rows[i] = testRow{id: name, cols: []string{name}}
				}
				m.SetRows(rows)
				if filter != "" {
					m.SetFilter(filter)
				}
				visible := m.RowCount()
				// The corners are what matter: top to bottom and back is the
				// widest jump the view can ask for.
				for _, pair := range [][2]int{{0, visible - 1}, {visible - 1, 0}, {0, visible / 2}, {visible - 1, visible / 2}} {
					from, to := pair[0], pair[1]
					if from < 0 || to < 0 {
						continue
					}
					m.SetCursor(from)
					m.SetCursor(to)
					want := m.filtered[to].ID()
					if !strings.Contains(ansi.Strip(m.View()), want) {
						t.Errorf("height %d, %d rows, filter %q: after %d -> %d the cursor's row %s is off screen",
							height, rowCount, filter, from, to, want)
					}
					checked++
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("the grid checked nothing")
	}
}

// The cursor's row is found by the escape sequence the Selected style opens
// it with, matched as a line prefix rather than searched for anywhere in the
// render: cell content arrives pre-styled from several views, and a cell
// carrying the same sequence must not answer for the cursor. Asserted here
// on hand-built lines, because lipgloss re-emits the sequences it is given,
// so a poisoned cell fed through the real render is inert.
func TestSelectedRowVisible_MatchesOnlyAtTheStartOfALine(t *testing.T) {
	const marker = "\x1b[7m"
	cases := []struct {
		name  string
		lines []string
		want  bool
	}{
		{"marker opens a line", []string{"NAME", marker + "web"}, true},
		{"marker inside a cell", []string{"NAME", "web " + marker + "unhealthy"}, false},
		{"no marker at all", []string{"NAME", "web"}, false},
		{"no lines", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectedRowVisible(tc.lines, marker); got != tc.want {
				t.Errorf("selectedRowVisible(%q) = %v, want %v", tc.lines, got, tc.want)
			}
		})
	}
}

// --- Column gutters --------------------------------------------------------
//
// bubbles' table pads every cell to exactly its column width and joins the
// cells with no separator, so the only thing keeping one column's text off
// the next one's is the cell content being narrower than the column. Fixed
// columns get that gutter from calculateColumnWidths, which allocates
// Width+DefaultColumnSpacing; proportional columns are allocated a raw share
// of the remaining space, so a cell that filled it used to butt against its
// neighbour with no space at all.

// visibleLines strips the styling and returns the rendered rows: the header
// line, then one line per row.
func visibleLines(t *testing.T, table TableModel) []string {
	t.Helper()
	var out []string
	for _, line := range strings.Split(table.View(), "\n") {
		out = append(out, strings.TrimRight(ansi.Strip(line), " "))
	}
	return out
}

// columnStart returns the on-screen column, in cells rather than bytes, at
// which text first appears in line, and reports whether the cell before it
// is a space. Byte offsets are useless here: a fitted cell ends in "…",
// which is three bytes wide and one cell wide.
func columnStart(t *testing.T, line, text string) (int, bool) {
	t.Helper()
	byteIdx := strings.Index(line, text)
	if byteIdx < 0 {
		t.Fatalf("expected %q in %q", text, line)
	}
	cells := ansi.StringWidth(line[:byteIdx])
	if cells == 0 {
		t.Fatalf("expected %q not to start the line in %q", text, line)
	}
	prev := []rune(line[:byteIdx])
	return cells, prev[len(prev)-1] == ' '
}

func TestTableModel_ProportionalCellsKeepAGutter(t *testing.T) {
	cols := []Column{{Title: "A"}, {Title: "B"}}
	table := NewTableModel(cols)
	table.SetSize(40, 10)
	table.SetRows([]TableRow{testRow{id: "1", cols: []string{
		strings.Repeat("a", 60), strings.Repeat("b", 60),
	}}})

	row := visibleLines(t, table)[1]
	if _, spaced := columnStart(t, row, "bbb"); !spaced {
		t.Fatalf("expected a space between the two columns, got %q", row)
	}
}

// The same must hold for the header titles: at narrow widths a title is
// wider than its column ("IMAGE/DRIVER" in 10 columns of space), and a
// truncated title that fills the column butts the next title.
func TestTableModel_ProportionalHeadersKeepAGutter(t *testing.T) {
	cols := []Column{{Title: "IMAGE/DRIVER"}, {Title: "PORTS"}}
	table := NewTableModel(cols)
	table.SetSize(20, 10)
	table.SetRows([]TableRow{testRow{id: "1", cols: []string{"x", "y"}}})

	header := visibleLines(t, table)[0]
	if _, spaced := columnStart(t, header, "PORTS"); !spaced {
		t.Fatalf("expected a space between the two titles, got %q", header)
	}
}

// A fixed column's gutter is allocated as part of its width, so its content
// must stay inside Width and never spill into the spacing.
func TestTableModel_FixedCellsStayInsideTheirWidth(t *testing.T) {
	cols := []Column{{Title: "A", Width: 5, Fixed: true}, {Title: "B"}}
	table := NewTableModel(cols)
	table.SetSize(40, 10)
	table.SetRows([]TableRow{testRow{id: "1", cols: []string{
		strings.Repeat("a", 20), strings.Repeat("b", 20),
	}}})

	row := visibleLines(t, table)[1]
	start, spaced := columnStart(t, row, "bbb")
	if start != 5+DefaultColumnSpacing {
		t.Fatalf("expected the second column to start at %d, got %d in %q",
			5+DefaultColumnSpacing, start, row)
	}
	if !spaced {
		t.Fatalf("expected a space between the two columns, got %q", row)
	}
}

// The last column has nothing to its right, so it keeps its full width: a
// gutter there would throw away a character of PORTS for no gain.
func TestTableModel_LastColumnUsesItsFullWidth(t *testing.T) {
	cols := []Column{{Title: "A", Width: 10, Fixed: true}, {Title: "B"}}
	table := NewTableModel(cols)
	table.SetSize(30, 10)
	table.SetRows([]TableRow{testRow{id: "1", cols: []string{
		"a", strings.Repeat("b", 60),
	}}})

	row := visibleLines(t, table)[1]
	if got := ansi.StringWidth(row); got != 30 {
		t.Fatalf("expected the last column to fill the table width, got %d in %q", got, row)
	}
}

// Cells are fitted to the column widths, so a resize has to re-fit them: a
// row set while the table was narrow must not stay truncated once the
// terminal grows, and must not spill once it shrinks.
func TestTableModel_ResizeRefitsCells(t *testing.T) {
	cols := []Column{{Title: "A"}, {Title: "B"}}
	table := NewTableModel(cols)
	table.SetSize(20, 10)
	table.SetRows([]TableRow{testRow{id: "1", cols: []string{
		strings.Repeat("a", 60), strings.Repeat("b", 60),
	}}})
	narrow := strings.Count(visibleLines(t, table)[1], "a")

	table.SetSize(80, 10)
	wide := strings.Count(visibleLines(t, table)[1], "a")
	if wide <= narrow {
		t.Fatalf("expected the widened table to show more of the cell, got %d then %d", narrow, wide)
	}

	table.SetSize(20, 10)
	row := visibleLines(t, table)[1]
	if got := strings.Count(row, "a"); got != narrow {
		t.Fatalf("expected the narrowed table to fit the cell again, got %d (want %d) in %q", got, narrow, row)
	}
	if _, spaced := columnStart(t, row, "bbb"); !spaced {
		t.Fatalf("expected a space between the two columns after narrowing, got %q", row)
	}
}

// Fitting must not destroy the styling a cell arrives with: the container
// list's status cell is pre-colored, and truncating it has to keep the escape
// sequences balanced or the color bleeds into the rest of the line.
func TestTableModel_FittingKeepsStyledCellsBalanced(t *testing.T) {
	cols := []Column{{Title: "A"}, {Title: "B"}}
	table := NewTableModel(cols)
	table.SetSize(24, 10)
	styled := ColorFg(strings.Repeat("a", 40), DryTheme.Success)
	table.SetRows([]TableRow{testRow{id: "1", cols: []string{styled, strings.Repeat("b", 40)}}})

	rendered := strings.Split(table.View(), "\n")[1]
	row := strings.TrimRight(ansi.Strip(rendered), " ")
	if _, spaced := columnStart(t, row, "bbb"); !spaced {
		t.Fatalf("expected a space between the two columns, got %q", row)
	}
	// The truncated cell's own color must be closed before the gutter, or
	// it bleeds across the gap and into the next column. Walk the SGR
	// state to the gutter cell and require nothing of the cell's own to be
	// active there. The row is the cursor row, so the selected-row
	// background is expected and ignored.
	fg, bg := sgrStateAt(rendered, table.contentWidths[0])
	if fg != "" {
		t.Fatalf("expected no foreground active in the gutter, got %q in %q", fg, rendered)
	}
	if bg == "" {
		t.Fatalf("expected the selected-row background still active in the gutter, got %q", rendered)
	}
}

// sgrStateAt replays the SGR sequences in a rendered line up to the given
// screen cell and reports the foreground and background active there, as
// the raw parameter strings, so an unbalanced opener is visible in the
// failure message.
func sgrStateAt(line string, cell int) (fgOut, bgOut string) {
	var fg, bg string
	pos := 0
	rest := line
	for pos <= cell && rest != "" {
		if strings.HasPrefix(rest, "\x1b[") {
			end := strings.Index(rest, "m")
			if end < 0 {
				break
			}
			params := rest[2:end]
			switch {
			case params == "" || params == "0":
				fg, bg = "", ""
			case strings.HasPrefix(params, "38;"):
				fg = params
			case params == "39":
				fg = ""
			case strings.HasPrefix(params, "48;"):
				bg = params
			case params == "49":
				bg = ""
			}
			// A combined sequence carries both.
			if i := strings.Index(params, ";48;"); i > 0 {
				bg = params[i+1:]
			}
			rest = rest[end+1:]
			continue
		}
		r, size := utf8.DecodeRuneInString(rest)
		pos += ansi.StringWidth(string(r))
		rest = rest[size:]
	}
	return fg, bg
}

// TestTableModel_EveryColumnKeepsItsGutterAtEveryWidth is the gate test for
// the gutter: it sweeps every width a terminal is likely to have, against
// the column sets the app actually ships, and checks the cell before each
// column boundary is a space. Width-dependent layout has bands, a
// proportional column lands at one or two cells wide when the fixed columns
// nearly fill the terminal, and a test at three hand-picked widths walks
// straight past them.
func TestTableModel_EveryColumnKeepsItsGutterAtEveryWidth(t *testing.T) {
	// The real column sets, read from the models that ship them, plus one
	// synthetic shape the app has no example of: a proportional column
	// first and a fixed one last.
	containers := NewContainersModel()
	// The compact set is what workspace mode swaps in, in the narrow pane
	// where the allocation is tightest, so it belongs in a sweep claiming
	// to cover the sets the app ships.
	compact := containerColumns(true)
	images := NewImagesModel()
	networks := NewNetworksModel()
	volumes := NewVolumesModel()
	monitor := NewMonitorModel()
	sets := map[string][]Column{
		"containers":         containers.table.columns,
		"images":             images.table.columns,
		"networks":           networks.table.columns,
		"volumes":            volumes.table.columns,
		"monitor":            monitor.table.columns,
		"containers compact": compact,
		"fixed last": {
			{Title: "NAME"},
			{Title: "SIZE", Width: 10, Fixed: true},
		},
	}

	for name, cols := range sets {
		t.Run(name, func(t *testing.T) {
			cells := make([]string, len(cols))
			for i := range cells {
				// Long enough to fill any column at any width, ASCII so a
				// rune index is a screen cell.
				cells[i] = strings.Repeat(string(rune('a'+i)), 80)
			}
			for width := 20; width <= 210; width++ {
				table := NewTableModel(cols)
				table.SetSize(width, 12)
				table.SetRows([]TableRow{testRow{id: "1", cols: cells}})

				lines := visibleLinesPadded(t, table)
				boundary := 0
				for i := 0; i < len(cols)-1; i++ {
					alloc := table.colWidths[i]
					boundary += alloc
					if boundary >= width {
						break // this boundary is off the right edge
					}
					if alloc < 2 {
						// One cell holds the ellipsis and nothing else;
						// see TestTableModel_FittingIsNeverTradedForSpacing.
						continue
					}
					for row, line := range lines {
						runes := []rune(line)
						if len(runes) < boundary {
							t.Fatalf("%s width %d: line %d is %d cells, expected at least %d",
								name, width, row, len(runes), boundary)
						}
						if runes[boundary-1] != ' ' {
							t.Fatalf("%s width %d: column %d butts column %d on line %d:\n%s",
								name, width, i, i+1, row, line)
						}
					}
				}
			}
		})
	}
}

// visibleLinesPadded returns the header line and the first row, styling
// stripped and padding kept, so a rune index is a screen cell.
func visibleLinesPadded(t *testing.T, table TableModel) []string {
	t.Helper()
	lines := strings.Split(table.View(), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected a header and a row, got %d lines", len(lines))
	}
	return []string{ansi.Strip(lines[0]), ansi.Strip(lines[1])}
}

// A proportional column wide enough for it shows content, the ellipsis that
// says the text was cut, and the gutter. Three cells is where that starts:
// the gutter takes one and the ellipsis takes one, so a two-cell column
// shows a bare ellipsis and still keeps its gutter, and only a one-cell
// column has nothing left for one. Narrowing further is what the allocator
// prefers to dropping a column outright.
func TestTableModel_ProportionalColumnsKeepRoomToTrim(t *testing.T) {
	cols := []Column{
		{Title: "A", Width: 40, Fixed: true},
		{Title: "NAME"},
		{Title: "B", Width: 10, Fixed: true},
	}
	for width := 44; width <= 60; width++ {
		table := NewTableModel(cols)
		table.SetSize(width, 10)
		table.SetRows([]TableRow{testRow{id: "1", cols: []string{
			"aaaaaaaaaa", strings.Repeat("n", 40), "bbbbbbbbbb",
		}}})

		row := visibleLinesPadded(t, table)[1]
		runes := []rune(row)
		start := table.colWidths[0]
		if start+table.colWidths[1] > width {
			continue // the column is clipped by the right edge at this width
		}
		cell := strings.TrimRight(string(runes[start:min(start+table.colWidths[1], len(runes))]), " ")
		if table.colWidths[1] < 3 {
			// Two cells hold the ellipsis and the gutter, one holds the
			// ellipsis alone: content needs a third.
			if cell != "…" {
				t.Fatalf("width %d: expected a bare ellipsis in a one-cell column, got %q", width, cell)
			}
			continue
		}
		if !strings.Contains(cell, "n") {
			t.Fatalf("width %d: the fitted cell shows no content at all: %q (row %q)",
				width, cell, row)
		}
		if !strings.HasSuffix(cell, "…") {
			t.Fatalf("width %d: expected the cut to be marked, got %q", width, cell)
		}
	}
}

// A column set with no proportional column is the one case where the last
// column is rendered wider than it was allocated: syncInnerColumns stretches
// it to fill the table. Its text must be fitted to what it is rendered at,
// not to the allocation, or most of the column comes out blank with the
// content cut short at its left edge.
func TestTableModel_AllFixedColumnsKeepTheirLastColumnsText(t *testing.T) {
	table := NewTableModel([]Column{
		{Title: "A", Width: 6, Fixed: true},
		{Title: "B", Width: 6, Fixed: true},
	})
	table.SetSize(80, 8)
	table.SetRows([]TableRow{testRow{id: "1", cols: []string{
		"aaa", strings.Repeat("b", 60),
	}}})

	row := visibleLinesPadded(t, table)[1]
	if got := strings.Count(row, "b"); got < 60 {
		t.Fatalf("expected the stretched last column to keep its text, got %d of 60 in %q", got, row)
	}
}

// Rows can be set before the table has ever been sized, a view that loads
// its data before the first window-size message. bubbles' table indexes its
// column slice per cell while rendering, so handing it rows with no columns
// panics; the rows wait for the size instead.
func TestTableModel_RowsBeforeTheFirstSizeDoNotPanic(t *testing.T) {
	table := NewTableModel([]Column{{Title: "NAME"}, {Title: "SIZE", Width: 8, Fixed: true}})
	table.SetRows([]TableRow{
		testRow{id: "1", cols: []string{"one", "1kB"}},
		testRow{id: "2", cols: []string{"two", "2kB"}},
	})
	if got := table.RowCount(); got != 2 {
		t.Fatalf("expected the rows to be kept, got %d", got)
	}
	if view := table.View(); view != "" {
		t.Fatalf("expected an unsized table to render nothing, got %q", view)
	}

	table.SetSize(40, 6)
	if !strings.Contains(ansi.Strip(table.View()), "one") {
		t.Fatalf("expected the rows once sized, got:\n%s", ansi.Strip(table.View()))
	}
	if got := table.Cursor(); got != 0 {
		t.Fatalf("expected the cursor on the first row, got %d", got)
	}
}

// Spacing must never cost a column: the columns overflow only when one cell
// each does not fit, and then by the least the shortfall allows. An earlier
// revision floored proportional columns at two cells plus the gutter, which
// overflowed these columns to 67 at every width from 59 to 66 and dropped
// PORTS at widths where it had fitted.
func TestTableModel_FittingIsNeverTradedForSpacing(t *testing.T) {
	cols := []Column{
		{Title: "NAME"},
		{Title: "CONTAINERS", Width: 12, Fixed: true},
		{Title: "RUNNING", Width: 10, Fixed: true},
		{Title: "EXITED", Width: 10, Fixed: true},
		{Title: "IMAGE/DRIVER"},
		{Title: "HEALTH/SCOPE", Width: 14, Fixed: true},
		{Title: "SYNC", Width: 7, Fixed: true},
		{Title: "PORTS"},
	}
	proportional, fixedTotal := 0, 0
	for _, c := range cols {
		if c.Fixed && c.Width > 0 {
			fixedTotal += c.Width + DefaultColumnSpacing
			continue
		}
		proportional++
	}
	for width := 20; width <= 250; width++ {
		table := NewTableModel(cols)
		table.SetSize(width, 12)

		total := 0
		for _, w := range table.colWidths {
			total += w
		}
		switch {
		case fixedTotal >= width:
			// The fixed columns alone do not fit, which is the older
			// minProportionalColumnWidth trade, not this one. The 88 is a
			// literal on purpose: 58 of fixed columns plus three
			// proportional ones at their 10-cell minimum. Deriving it from
			// minProportionalColumnWidth would move with the constant and
			// assert nothing.
			if total != 88 {
				t.Fatalf("width %d: expected the minimum-width allocation 88, got %d",
					width, total)
			}
		case fixedTotal+proportional <= width:
			// One cell each fits, so the whole allocation must fit.
			if total > width {
				t.Fatalf("width %d: columns total %d, overflowing by %d with room to fit",
					width, total, total-width)
			}
		default:
			// One cell each does not fit. The overflow is then exactly the
			// shortfall, never more.
			if want := fixedTotal + proportional; total != want {
				t.Fatalf("width %d: expected one cell per proportional column (%d), got %d",
					width, want, total)
			}
		}
	}
}

// The gutter a proportional column keeps is taken out of its content, so a
// column of two cells or more has one. A one-cell column cannot, and that
// is the case the allocator prefers over losing a column outright.
func TestTableModel_TwoCellColumnsKeepTheirGutter(t *testing.T) {
	InitStyles()
	cols := []Column{{Title: "A"}, {Title: "B"}, {Title: "C"}}
	for width := 6; width <= 40; width++ {
		table := NewTableModel(cols)
		table.SetSize(width, 6)
		table.SetRows([]TableRow{testRow{id: "1", cols: []string{
			strings.Repeat("a", 60), strings.Repeat("b", 60), strings.Repeat("c", 60),
		}}})
		lines := visibleLinesPadded(t, table)
		boundary := 0
		for i := 0; i < len(cols)-1; i++ {
			alloc := table.colWidths[i]
			boundary += alloc
			if boundary >= width || alloc < 2 {
				continue
			}
			for row, line := range lines {
				runes := []rune(line)
				if len(runes) < boundary {
					t.Fatalf("width %d: line %d is %d cells, expected %d", width, row, len(runes), boundary)
				}
				if runes[boundary-1] != ' ' {
					t.Fatalf("width %d: column %d (%d cells) butts its neighbour on line %d:\n%s",
						width, i, alloc, row, line)
				}
			}
		}
	}
}

// The gutter has to survive a cell that exactly fills its content limit, and
// one a single cell over it: the first takes the no-truncation path through
// fitCell, the second the ellipsis path, and only the second was covered by
// the sweeps, which all use cells far longer than any column.
func TestTableModel_TheGutterHoldsAtTheContentLimit(t *testing.T) {
	InitStyles()
	cols := []Column{{Title: "A"}, {Title: "B"}, {Title: "C"}}
	for width := 12; width <= 60; width++ {
		table := NewTableModel(cols)
		table.SetSize(width, 6)
		limit := table.contentWidths[0]
		if limit < 2 {
			continue
		}
		for _, length := range []int{limit - 1, limit, limit + 1} {
			table.SetRows([]TableRow{testRow{id: "1", cols: []string{
				strings.Repeat("a", length), "b", "c",
			}}})
			alloc := table.colWidths[0]
			for _, line := range visibleLinesPadded(t, table) {
				runes := []rune(line)
				if len(runes) < alloc || strings.TrimSpace(string(runes[:alloc])) == "" {
					continue
				}
				if runes[alloc-1] != ' ' {
					t.Fatalf("width %d, cell of %d against a limit of %d: no gutter left:\n%s",
						width, length, limit, line)
				}
			}
		}
	}
}

// The reason goes where the first row would be, so the column titles still
// say what the columns are.
func TestTableModel_EmptyMessageGoesWhereTheFirstRowWouldBe(t *testing.T) {
	InitStyles()
	m := NewTableModel([]Column{{Title: "NAME"}, {Title: "SIZE", Width: 8, Fixed: true}})
	m.SetSize(60, 8)
	m.SetEmptyMessage("Nothing here yet")
	m.SetRows(nil)

	lines := strings.Split(ansi.Strip(m.View()), "\n")
	if !strings.Contains(lines[0], "NAME") {
		t.Fatalf("expected the column titles on the first line, got %q", lines[0])
	}
	if got := strings.TrimRight(lines[1], " "); got != "Nothing here yet" {
		t.Errorf("expected the message on the second line, got %q", got)
	}
	if got := ansi.StringWidth(strings.Split(m.View(), "\n")[1]); got != 60 {
		t.Errorf("expected the message line padded to the width, got %d", got)
	}
	// Muted, not the colour of a row: it is an explanation in the place a
	// row would be, and at full strength it reads as data.
	styled := strings.Split(m.View(), "\n")[1]
	if want := ColorFg("Nothing here yet", DryTheme.FgMuted); !strings.Contains(styled, want) {
		t.Errorf("expected the message rendered muted, got %q", styled)
	}
}

// With rows to show, the message must not appear.
func TestTableModel_EmptyMessageIsOnlyForAnEmptyTable(t *testing.T) {
	InitStyles()
	m := NewTableModel([]Column{{Title: "NAME"}})
	m.SetSize(40, 8)
	m.SetEmptyMessage("Nothing here yet")
	m.SetRows([]TableRow{testRow{id: "1", cols: []string{"one"}}})

	if view := ansi.Strip(m.View()); strings.Contains(view, "Nothing here yet") {
		t.Errorf("expected no empty-state line with a row present:\n%s", view)
	}
}

// A message longer than the terminal is cut mid-word, like any other
// clipped line, but the ellipsis says so rather than leaving it looking
// like a rendering fault.
func TestTableModel_ALongEmptyMessageIsMarkedWhenCut(t *testing.T) {
	InitStyles()
	m := NewTableModel([]Column{{Title: "NAME"}})
	m.SetSize(30, 6)
	m.SetEmptyMessage(strings.Repeat("a very long explanation ", 4))
	m.SetRows(nil)

	line := strings.TrimRight(ansi.Strip(strings.Split(m.View(), "\n")[1]), " ")
	if ansi.StringWidth(line) > 30 {
		t.Errorf("expected the message inside the width, got %d cells: %q", ansi.StringWidth(line), line)
	}
	if !strings.HasSuffix(line, "…") {
		t.Errorf("expected the cut marked, got %q", line)
	}
}

// SetSortField assigns -1 for a view whose rows are sorted elsewhere, so a
// grouped model can reach the comparison with that field. Every cell then
// reads as empty, which is an order, not a panic.
func TestCompareRowsByColumn_AColumnOutsideTheRow(t *testing.T) {
	a := testRow{id: "a", cols: []string{"aaa", "1"}}
	b := testRow{id: "b", cols: []string{"bbb", "2"}}

	for _, col := range []int{-1, 2, 99} {
		if CompareRowsByColumn(a, b, col, true) || CompareRowsByColumn(b, a, col, true) {
			t.Errorf("column %d: expected the rows to compare equal, not ordered", col)
		}
	}
	if !CompareRowsByColumn(a, b, 0, true) {
		t.Error("column 0: expected aaa before bbb")
	}
}

// NextSortField moves the indicator one column on and wraps, and it does
// not sort: a grouped model calls it and rebuilds its rows in order
// instead. Off by one, one press per cycle lands on -1, which SetSortField
// reads as "sorted elsewhere" and which stops the indicator rendering.
func TestTableModel_NextSortFieldCyclesWithoutSorting(t *testing.T) {
	InitStyles()
	m := NewTableModel([]Column{{Title: "NAME"}, {Title: "SIZE"}, {Title: "AGE"}})
	m.SetSize(60, 8)
	rows := []TableRow{
		testRow{id: "b", cols: []string{"bbb", "2", "x"}},
		testRow{id: "a", cols: []string{"aaa", "1", "y"}},
	}
	m.SetRows(rows)

	// Two turns of the cycle, so a wrap that lands off the end shows.
	for turn := range 2 {
		for want := range 3 {
			if got := m.SortField(); got != want {
				t.Fatalf("turn %d: expected field %d, got %d", turn, want, got)
			}
			if !strings.Contains(ansi.Strip(m.View()), m.columns[want].Title+" ↓") {
				t.Errorf("turn %d: expected the indicator on %s", turn, m.columns[want].Title)
			}
			m.NextSortField()
		}
	}

	// And the rows are where they were put: this moves the indicator only.
	if first := m.FilteredRows()[0].ID(); first != "b" {
		t.Errorf("expected the rows left in the order they were set, got %s first", first)
	}
}
