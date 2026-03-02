package appui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
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
