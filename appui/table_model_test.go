package appui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
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

	// Move down with arrow
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

	// Verify visible range includes cursor
	start, end := table.visibleRange()
	if table.Cursor() < start || table.Cursor() >= end {
		t.Fatalf("cursor %d outside visible range [%d, %d)", table.Cursor(), start, end)
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
