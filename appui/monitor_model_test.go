package appui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/moncho/dry/docker"
)

func TestMonitorBar_VisualWidth(t *testing.T) {
	// Bar visual width should always equal barWidth, and the full output
	// (bar + label) should have a consistent visual width.
	const barWidth = 12
	for _, pct := range []float64{0, 0.3, 1, 5, 10, 25, 50, 75, 80, 90, 100} {
		bar := monitorBar(pct, barWidth)
		// Label is " %5.1f%%" = 7 chars. Total = barWidth + 7.
		want := barWidth + 7
		got := ansi.StringWidth(bar)
		if got != want {
			t.Errorf("monitorBar(%.1f, %d): visual width = %d, want %d",
				pct, barWidth, got, want)
		}
	}
}

func TestMonitorBar_FillProgression(t *testing.T) {
	const barWidth = 12

	countFilled := func(bar string) int {
		n := 0
		for _, r := range bar {
			if r == '█' || r == '▌' {
				n++
			}
		}
		return n
	}

	// 0% should have no fill characters.
	if n := countFilled(monitorBar(0, barWidth)); n != 0 {
		t.Errorf("monitorBar(0): filled chars = %d, want 0", n)
	}

	// Any value > 0 should show at least one fill character (half-block minimum).
	for _, pct := range []float64{0.1, 0.5, 1, 3} {
		if n := countFilled(monitorBar(pct, barWidth)); n < 1 {
			t.Errorf("monitorBar(%.1f): filled chars = %d, want >= 1", pct, n)
		}
	}

	// Higher percentage should have more fill than lower.
	lowFill := countFilled(monitorBar(10, barWidth))
	highFill := countFilled(monitorBar(50, barWidth))
	if highFill <= lowFill {
		t.Errorf("monitorBar(50) fill (%d) should be > monitorBar(10) fill (%d)",
			highFill, lowFill)
	}

	// 100% should fill the entire bar.
	if n := countFilled(monitorBar(100, barWidth)); n != barWidth {
		t.Errorf("monitorBar(100): filled chars = %d, want %d", n, barWidth)
	}
}

func TestMonitorBar_ColorThreshold(t *testing.T) {
	// At <= 80% the bar uses DryTheme.Info color.
	// Above 80% it switches to DryTheme.Warning.
	low := monitorBar(50, 12)
	high := monitorBar(90, 12)

	// The ANSI codes should differ because the colors differ.
	if low == high {
		t.Error("monitorBar(50) and monitorBar(90) should use different colors")
	}
}

func TestMonitorBar_Label(t *testing.T) {
	bar := monitorBar(42.7, 12)
	if !strings.Contains(bar, "42.7%") {
		t.Errorf("monitorBar(42.7) should contain label '42.7%%', got %q", bar)
	}
}

func TestMonitorBar_Clamping(t *testing.T) {
	// Values above 100 should be clamped.
	bar := monitorBar(150, 12)
	got := ansi.StringWidth(bar)
	want := 12 + 7
	if got != want {
		t.Errorf("monitorBar(150): visual width = %d, want %d", got, want)
	}
}

// newTestMonitor creates a MonitorModel with stats pre-populated
// and the table sized for testing.
func newTestMonitor(stats map[string]*docker.Stats) MonitorModel {
	m := NewMonitorModel()
	m.SetSize(200, 25)
	m.stats = stats
	m.refreshTable()
	return m
}

func TestMonitor_SortPreservedAcrossRefresh(t *testing.T) {
	stats := map[string]*docker.Stats{
		"ccc": {CID: "ccc", Command: "cmd_c", CPUPercentage: 30},
		"aaa": {CID: "aaa", Command: "cmd_a", CPUPercentage: 10},
		"bbb": {CID: "bbb", Command: "cmd_b", CPUPercentage: 20},
	}
	m := newTestMonitor(stats)

	// Default sort is column 0 (CONTAINER), ascending.
	first := m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected first row 'aaa' with default sort, got %v", first)
	}

	// Cycle sort to column 7 (COMMAND) via F1 presses.
	// Columns: 0=CONTAINER, 1=CPU%, 2=MEM USAGE, 3=MEM%, 4=NET, 5=BLOCK, 6=PIDS, 7=COMMAND
	for range 7 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	}
	if m.table.SortField() != 7 {
		t.Fatalf("expected sort field 7 (COMMAND), got %d", m.table.SortField())
	}

	// After sorting by COMMAND, first row should be cmd_a
	first = m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected first row 'aaa' sorted by COMMAND, got %v", first)
	}

	// Simulate stats update — this calls refreshTable() internally.
	// The sort should be preserved.
	stats["aaa"].CPUPercentage = 99
	m.stats = stats
	m.refreshTable()

	first = m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected first row 'aaa' after refreshTable, got %v", first)
	}
}

func TestMonitor_SortChangesRowOrder(t *testing.T) {
	stats := map[string]*docker.Stats{
		"zzz": {CID: "zzz", Command: "alpha"},
		"aaa": {CID: "aaa", Command: "zulu"},
		"mmm": {CID: "mmm", Command: "mike"},
	}
	m := newTestMonitor(stats)

	// Default sort by column 0 (CONTAINER) ascending → aaa first
	first := m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected first row 'aaa' sorted by CONTAINER, got %v", first)
	}

	// Sort by COMMAND (column 7) → "alpha" (zzz) first
	for range 7 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	}

	first = m.table.SelectedRow()
	if first == nil || first.ID() != "zzz" {
		t.Fatalf("expected first row 'zzz' sorted by COMMAND, got %v", first)
	}
}

func TestMonitor_RefreshWithNewContainer(t *testing.T) {
	stats := map[string]*docker.Stats{
		"bbb": {CID: "bbb", Command: "beta"},
		"aaa": {CID: "aaa", Command: "alpha"},
	}
	m := newTestMonitor(stats)

	// Sort by CONTAINER (default) — aaa first
	first := m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected 'aaa' first, got %v", first)
	}

	// Add a new container and refresh
	m.stats["ccc"] = &docker.Stats{CID: "ccc", Command: "charlie"}
	m.refreshTable()

	// aaa should still be first (sorted by CONTAINER)
	first = m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected 'aaa' first after adding ccc, got %v", first)
	}
	if m.table.RowCount() != 3 {
		t.Fatalf("expected 3 rows, got %d", m.table.RowCount())
	}
}
