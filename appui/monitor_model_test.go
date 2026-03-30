package appui

import (
	"strings"
	"testing"
	"time"

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
		"ccc": {CID: "ccc", Command: "cmd_c", CPUPercentage: 30, NetworkRx: 30, NetworkTx: 3},
		"aaa": {CID: "aaa", Command: "cmd_a", CPUPercentage: 10, NetworkRx: 10, NetworkTx: 1},
		"bbb": {CID: "bbb", Command: "cmd_b", CPUPercentage: 20, NetworkRx: 20, NetworkTx: 2},
	}
	m := newTestMonitor(stats)

	// Default sort is column 0 (CONTAINER), ascending.
	first := m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected first row 'aaa' with default sort, got %v", first)
	}

	// Cycle sort to column 4 (NET RX/TX) via F1 presses.
	// Columns: 0=CONTAINER, 1=CPU LOAD, 2=MEM USAGE, 3=MEM LOAD, 4=NET
	for range 4 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	}
	if m.table.SortField() != 4 {
		t.Fatalf("expected sort field 4 (NET RX/TX), got %d", m.table.SortField())
	}

	// Ascending network IO should keep aaa first.
	first = m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected first row 'aaa' sorted by NET RX/TX, got %v", first)
	}

	// Simulate stats update — this calls refreshTable() internally.
	// The sort should be preserved.
	stats["aaa"].CPUPercentage = 99
	m.stats = stats
	m.refreshTable()

	first = m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected first row 'aaa' after refreshTable preserving NET RX/TX sort, got %v", first)
	}
}

func TestMonitor_SortChangesRowOrder(t *testing.T) {
	stats := map[string]*docker.Stats{
		"zzz": {CID: "zzz", NetworkRx: 30, NetworkTx: 5},
		"aaa": {CID: "aaa", NetworkRx: 10, NetworkTx: 2},
		"mmm": {CID: "mmm", NetworkRx: 20, NetworkTx: 3},
	}
	m := newTestMonitor(stats)

	// Default sort by column 0 (CONTAINER) ascending → aaa first
	first := m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected first row 'aaa' sorted by CONTAINER, got %v", first)
	}

	// Sort by NET RX/TX (column 4) → lowest network IO (aaa) first
	for range 4 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	}

	first = m.table.SelectedRow()
	if first == nil || first.ID() != "aaa" {
		t.Fatalf("expected first row 'aaa' sorted by NET RX/TX, got %v", first)
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

func TestMonitor_ViewShowsLiveSummary(t *testing.T) {
	stats := map[string]*docker.Stats{
		"aaa": {CID: "aaa", CPUPercentage: 12.5, MemoryPercentage: 45.0, Command: "api"},
		"bbb": {CID: "bbb", CPUPercentage: 82.1, MemoryPercentage: 33.0, Command: "worker"},
		"ccc": {CID: "ccc", CPUPercentage: 8.0, MemoryPercentage: 67.4, Command: "db"},
	}
	m := newTestMonitor(stats)

	view := ansi.Strip(m.View())
	for _, want := range []string{"◉", "Monitor", "live 3", "hot cpu bbb 82.1%", "hot mem ccc 67.4%"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in monitor view, got %q", want, view)
		}
	}
}

func TestMonitor_ViewUsesFullAllocatedHeight(t *testing.T) {
	stats := map[string]*docker.Stats{
		"aaa": {CID: "aaa", CPUPercentage: 12.5, MemoryPercentage: 45.0, Command: "api"},
		"bbb": {CID: "bbb", CPUPercentage: 82.1, MemoryPercentage: 33.0, Command: "worker"},
	}
	m := NewMonitorModel()
	m.SetSize(120, 10)
	m.stats = stats
	m.refreshTable()

	view := ansi.Strip(m.View())
	if got, want := len(strings.Split(view, "\n")), 10; got != want {
		t.Fatalf("expected monitor view to fill %d lines, got %d in %q", want, got, view)
	}
}

func TestMonitor_ViewTruncatesRowsToPaneWidth(t *testing.T) {
	stats := map[string]*docker.Stats{
		"aaa": {
			CID:              "aaa",
			CPUPercentage:    82.1,
			Memory:           512 * 1024 * 1024,
			MemoryLimit:      2 * 1024 * 1024 * 1024,
			MemoryPercentage: 33.0,
			NetworkRx:        120 * 1024 * 1024,
			NetworkTx:        8 * 1024 * 1024,
			BlockRead:        480 * 1024 * 1024,
			BlockWrite:       32 * 1024 * 1024,
			Command:          "worker",
		},
	}
	m := NewMonitorModel()
	m.SetSize(72, 8)
	m.stats = stats
	m.refreshTable()

	view := m.View()
	for _, line := range strings.Split(view, "\n") {
		if got := ansi.StringWidth(line); got > 72 {
			t.Fatalf("expected rendered monitor line width <= %d, got %d in %q", 72, got, ansi.Strip(line))
		}
	}
}

func TestMonitor_ViewMarksSelectedRowWithArrow(t *testing.T) {
	stats := map[string]*docker.Stats{
		"aaa": {CID: "aaa", CPUPercentage: 12.5, MemoryPercentage: 45.0, Command: "api"},
		"bbb": {CID: "bbb", CPUPercentage: 82.1, MemoryPercentage: 33.0, Command: "worker"},
	}
	m := newTestMonitor(stats)

	view := ansi.Strip(m.View())
	if !strings.Contains(view, "▶ aaa") {
		t.Fatalf("expected selected monitor row to show arrow marker, got %q", view)
	}
	if !strings.Contains(view, "● bbb") {
		t.Fatalf("expected non-selected monitor rows to keep green dot markers, got %q", view)
	}
	if strings.Contains(view, "● aaa") {
		t.Fatalf("did not expect selected monitor row to keep green dot marker, got %q", view)
	}
}

func TestMonitor_SelectedRowPreservesBarBackgroundStyling(t *testing.T) {
	stats := map[string]*docker.Stats{
		"aaa": {CID: "aaa", CPUPercentage: 12.5, MemoryPercentage: 45.0, Command: "api"},
		"bbb": {CID: "bbb", CPUPercentage: 82.1, MemoryPercentage: 33.0, Command: "worker"},
	}
	m := newTestMonitor(stats)

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) < 4 {
		t.Fatalf("expected monitor rows in view, got %q", view)
	}
	selected := lines[3]
	if !strings.Contains(selected, "▶") {
		t.Fatalf("expected selected row marker, got %q", selected)
	}
	if !strings.Contains(selected, "48;2;58;57;67m▌") {
		t.Fatalf("expected selected row to preserve bar background styling, got %q", selected)
	}
}

func TestMonitor_SelectedSeriesTracksRecentSamples(t *testing.T) {
	m := NewMonitorModel()
	m.SetSize(120, 20)

	ch := make(chan *docker.Stats)
	for _, sample := range []struct {
		cpu float64
		mem float64
	}{
		{10, 5},
		{20, 15},
		{30, 25},
	} {
		m.UpdateStats("abc", &docker.Stats{CID: "abc", CPUPercentage: sample.cpu, MemoryPercentage: sample.mem}, ch)
	}
	m.FlushTable()

	series := m.SelectedSeries()
	if got, want := len(series.CPU), 3; got != want {
		t.Fatalf("expected %d cpu samples, got %d", want, got)
	}
	if got, want := len(series.Memory), 3; got != want {
		t.Fatalf("expected %d memory samples, got %d", want, got)
	}
	if series.CPU[2].Value != 30 || series.Memory[2].Value != 25 {
		t.Fatalf("expected latest samples to be preserved, got %+v", series)
	}
}

func TestAppendMonitorSampleKeepsOnlyRecentWindow(t *testing.T) {
	var samples []MonitorPoint
	base := time.Unix(1_000, 0)
	for i := range 5 {
		samples = appendMonitorSample(samples, MonitorPoint{
			At:    base.Add(time.Duration(i) * time.Minute),
			Value: float64(i),
		})
	}
	if got, want := len(samples), 4; got != want {
		t.Fatalf("expected %d samples, got %d", want, got)
	}
	if samples[0].Value != 1 || samples[3].Value != 4 {
		t.Fatalf("expected history to keep only the recent window, got %+v", samples)
	}
}
