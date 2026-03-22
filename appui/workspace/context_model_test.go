package workspace

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestContextModelView_HidesItemCountInHeader(t *testing.T) {
	m := NewContextModel()
	m.SetSize(60, 8)
	m.SetMode("preview")
	m.SetContent("nginx:latest", "", []string{"size: 123MB"})

	view := ansi.Strip(m.View())
	header, _, _ := strings.Cut(view, "\n")

	if strings.Contains(header, "0") {
		t.Fatalf("expected context header to hide item counts, got %q", header)
	}
	if !strings.Contains(header, "Context · preview") {
		t.Fatalf("expected context header title, got %q", header)
	}
}

func TestContextModelView_ScrollsWithNavigationKeys(t *testing.T) {
	m := NewContextModel()
	m.SetSize(40, 5)
	m.SetMode("preview")
	m.SetContent("api", "", []string{
		"line 1", "line 2", "line 3", "line 4", "line 5", "line 6",
	})

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := m.viewport.YOffset(); got == 0 {
		t.Fatalf("expected context viewport to scroll down, got offset %d", got)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G'})
	if got := m.viewport.YOffset(); got == 0 {
		t.Fatalf("expected G to jump context viewport downward, got offset %d", got)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'g'})
	if got := m.viewport.YOffset(); got != 0 {
		t.Fatalf("expected g to jump context viewport to top, got offset %d", got)
	}
}

func TestContextModelSetContent_DoesNotResetViewportWhenContentIsUnchanged(t *testing.T) {
	m := NewContextModel()
	m.SetSize(40, 5)
	lines := []string{"line 1", "line 2", "line 3", "line 4", "line 5", "line 6"}
	m.SetMode("preview")
	m.SetContent("api", "", lines)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	offset := m.viewport.YOffset()
	if offset == 0 {
		t.Fatal("expected viewport to scroll before reapplying content")
	}

	m.SetContent("api", "", lines)
	if got := m.viewport.YOffset(); got != offset {
		t.Fatalf("expected viewport offset %d to be preserved, got %d", offset, got)
	}
}

func TestContextModelSetContent_PreservesViewportWhenBodyChangesForSameItem(t *testing.T) {
	m := NewContextModel()
	m.SetSize(40, 5)
	m.SetMode("preview")
	m.SetContent("api", "", []string{
		"line 1", "line 2", "line 3", "line 4", "line 5", "line 6",
	})

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	offset := m.viewport.YOffset()
	if offset == 0 {
		t.Fatal("expected viewport to scroll before content update")
	}

	m.SetContent("api", "", []string{
		"line 1", "line 2*", "line 3", "line 4", "line 5", "line 6", "line 7",
	})

	if got := m.viewport.YOffset(); got != offset {
		t.Fatalf("expected viewport offset %d to be preserved across content refresh, got %d", offset, got)
	}
}

func TestContextModelSetContent_ResetsViewportWhenItemChanges(t *testing.T) {
	m := NewContextModel()
	m.SetSize(40, 5)
	m.SetMode("preview")
	m.SetContent("api", "", []string{
		"line 1", "line 2", "line 3", "line 4", "line 5", "line 6",
	})

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := m.viewport.YOffset(); got == 0 {
		t.Fatal("expected viewport to scroll before item change")
	}

	m.SetContent("worker", "", []string{
		"line a", "line b", "line c", "line d", "line e", "line f",
	})

	if got := m.viewport.YOffset(); got != 0 {
		t.Fatalf("expected viewport to reset for a new item, got %d", got)
	}
}
