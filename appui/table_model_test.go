package appui

import (
	"fmt"
	"strings"
	"testing"

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
	if got := table.colWidths[1]; got < minProportionalColumnWidth {
		t.Fatalf("expected proportional column width >= %d when fixed columns overflow, got %d",
			minProportionalColumnWidth, got)
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
