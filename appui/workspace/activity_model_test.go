package workspace

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestActivityModel_MonitorDetailsHeaderHidesFollowAndCounts(t *testing.T) {
	m := NewActivityModel()
	m.SetSize(80, 12)
	m.SetContent("Monitor Details: redis", "Live stats", "line1\nline2")

	view := ansi.Strip(m.View())
	header := strings.Split(view, "\n")[0]
	if strings.Contains(header, "follow") {
		t.Fatalf("did not expect follow in monitor header, got %q", header)
	}
	if strings.Contains(header, "2") {
		t.Fatalf("did not expect line counts in monitor header, got %q", header)
	}
	if !strings.Contains(header, "Monitor Details: redis") {
		t.Fatalf("expected monitor title in header, got %q", header)
	}
}

func TestActivityModel_MonitorDetailsViewTruncatesToHeight(t *testing.T) {
	m := NewActivityModel()
	m.SetSize(80, 12)

	// Create content taller than the body (12 - 2 = 10 lines for body).
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, "line")
	}
	m.SetContent("Monitor Details: redis", "Live stats", strings.Join(lines, "\n"))

	view := m.View()
	got := len(strings.Split(view, "\n"))
	if got != m.Height() {
		t.Fatalf("expected view to produce exactly %d lines, got %d", m.Height(), got)
	}
}

func TestActivityModel_MonitorDetailsIgnoreScrollKeys(t *testing.T) {
	m := NewActivityModel()
	m.SetSize(80, 12)
	m.SetContent("Monitor Details: redis", "Live stats", "line1\nline2")

	next, _ := m.Update(tea.KeyPressMsg{Code: 'G'})
	if next.viewport.YOffset() != 0 {
		t.Fatalf("expected static monitor details to ignore scroll keys, got offset %d", next.viewport.YOffset())
	}
}
