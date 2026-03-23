package appui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// --- PromptModel tests ---

func TestPromptModel_ConfirmY(t *testing.T) {
	p := NewPromptModel("Delete?", "rm", "abc123")
	_, cmd := p.Update(tea.KeyPressMsg{Code: 'y'})
	if cmd == nil {
		t.Fatal("expected cmd from y")
	}
	msg := cmd()
	pr, ok := msg.(PromptResultMsg)
	if !ok {
		t.Fatalf("expected PromptResultMsg, got %T", msg)
	}
	if !pr.Confirmed {
		t.Fatal("expected confirmed")
	}
	if pr.Tag != "rm" || pr.ID != "abc123" {
		t.Fatalf("wrong tag/id: %q/%q", pr.Tag, pr.ID)
	}
}

func TestPromptModel_DenyN(t *testing.T) {
	p := NewPromptModel("Delete?", "rm", "abc123")
	_, cmd := p.Update(tea.KeyPressMsg{Code: 'n'})
	if cmd == nil {
		t.Fatal("expected cmd from n")
	}
	msg := cmd()
	pr := msg.(PromptResultMsg)
	if pr.Confirmed {
		t.Fatal("expected not confirmed")
	}
}

func TestPromptModel_DenyEsc(t *testing.T) {
	p := NewPromptModel("Delete?", "rm", "abc123")
	_, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cmd from esc")
	}
	msg := cmd()
	pr := msg.(PromptResultMsg)
	if pr.Confirmed {
		t.Fatal("expected not confirmed on esc")
	}
}

func TestPromptModel_IgnoresOtherKeys(t *testing.T) {
	p := NewPromptModel("Delete?", "rm", "abc123")
	_, cmd := p.Update(tea.KeyPressMsg{Code: 'x'})
	if cmd != nil {
		t.Fatal("expected nil cmd from unrecognized key")
	}
}

func TestPromptModel_View(t *testing.T) {
	p := NewPromptModel("Delete container?", "rm", "abc123")
	p.SetWidth(80)
	v := p.View()
	if v == "" {
		t.Fatal("View() should not be empty")
	}
}

// --- LessModel tests ---

func TestLessModel_SetContent(t *testing.T) {
	m := NewLessModel()
	m.SetSize(80, 24)
	m.SetContent("line1\nline2\nline3", "Test")

	v := m.View()
	if v == "" {
		t.Fatal("View() should not be empty after SetContent")
	}
}

func TestLessModel_EscCloses(t *testing.T) {
	m := NewLessModel()
	m.SetSize(80, 24)
	m.SetContent("content", "Test")

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cmd from esc")
	}
	msg := cmd()
	if _, ok := msg.(CloseOverlayMsg); !ok {
		t.Fatalf("expected CloseOverlayMsg, got %T", msg)
	}
}

func TestLessModel_QCloses(t *testing.T) {
	m := NewLessModel()
	m.SetSize(80, 24)
	m.SetContent("content", "Test")

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Fatal("expected cmd from q")
	}
	msg := cmd()
	if _, ok := msg.(CloseOverlayMsg); !ok {
		t.Fatalf("expected CloseOverlayMsg, got %T", msg)
	}
}

func TestLessModel_SearchMode(t *testing.T) {
	m := NewLessModel()
	m.SetSize(80, 24)
	m.SetContent("hello world\nfoo bar\nhello again", "Test")

	// Enter search mode
	m, _ = m.Update(tea.KeyPressMsg{Code: '/'})
	if m.mode != lessSearching {
		t.Fatalf("expected lessSearching, got %d", m.mode)
	}

	// Escape cancels search
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != lessNormal {
		t.Fatalf("expected lessNormal after esc, got %d", m.mode)
	}
}

func TestLessModel_FilterMode(t *testing.T) {
	m := NewLessModel()
	m.SetSize(80, 24)
	m.SetContent("hello world\nfoo bar\nhello again", "Test")

	// Enter filter mode
	m, _ = m.Update(tea.KeyPressMsg{Code: 'F'})
	if m.mode != lessFiltering {
		t.Fatalf("expected lessFiltering, got %d", m.mode)
	}

	// Escape cancels filter
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != lessNormal {
		t.Fatalf("expected lessNormal after esc, got %d", m.mode)
	}
}

func TestLessModel_FollowToggle(t *testing.T) {
	m := NewLessModel()
	m.SetSize(80, 24)
	m.SetContent("line1\nline2", "Test")

	if m.following {
		t.Fatal("expected not following initially")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'f'})
	if !m.following {
		t.Fatal("expected following after f")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'f'})
	if m.following {
		t.Fatal("expected not following after second f")
	}
}

func TestQuickPeekModel_SpaceCloses(t *testing.T) {
	m := NewQuickPeekModel()
	m.SetSize(120, 40)

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if cmd == nil {
		t.Fatal("expected cmd from space")
	}
	msg := cmd()
	if _, ok := msg.(CloseOverlayMsg); !ok {
		t.Fatalf("expected CloseOverlayMsg, got %T", msg)
	}
}

func TestQuickPeekModel_View(t *testing.T) {
	m := NewQuickPeekModel()
	m.SetSize(120, 40)
	m.SetContent(
		"api-1",
		"Container",
		"Recent Logs",
		"Recent logs · last 128 lines",
		[]string{"status: running", "image: nginx:latest"},
		"line one\nline two",
	)

	v := m.View()
	if !strings.Contains(v, "Quick Peek") {
		t.Fatal("expected quick peek title")
	}
	if !strings.Contains(v, "api-1") {
		t.Fatal("expected selected item title")
	}
	if !strings.Contains(v, "Recent Logs") {
		t.Fatal("expected detail title")
	}
	if !strings.Contains(v, "────") {
		t.Fatal("expected visual divider between summary and preview")
	}
}

func TestQuickPeekModel_StaysNearTop(t *testing.T) {
	m := NewQuickPeekModel()
	m.SetSize(120, 40)
	m.SetContent(
		"api-1",
		"Container",
		"Recent Logs",
		"Recent logs · last 128 lines",
		[]string{"status: running", "image: nginx:latest"},
		"line one\nline two",
	)

	lines := strings.Split(m.View(), "\n")
	firstBoxLine := -1
	for i, line := range lines {
		if strings.Contains(line, "╭") {
			firstBoxLine = i
			break
		}
	}
	if firstBoxLine == -1 {
		t.Fatal("expected quick peek dialog border")
	}
	if firstBoxLine > 2 {
		t.Fatalf("expected quick peek to stay near the top, got top margin of %d lines", firstBoxLine)
	}
}

func TestQuickPeekModel_GAndGJump(t *testing.T) {
	m := NewQuickPeekModel()
	m.SetSize(120, 24)
	m.SetContent(
		"api-1",
		"Container",
		"Recent Logs",
		"Recent logs · last 128 lines",
		[]string{"status: running"},
		strings.Repeat("line\n", 80),
	)

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G'})
	if m.viewport.YOffset() == 0 {
		t.Fatal("expected G to jump to the bottom")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'g'})
	if m.viewport.YOffset() != 0 {
		t.Fatal("expected g to jump to the top")
	}
}

func TestQuickPeekModel_GKeepsStableHeightWithTrailingNewline(t *testing.T) {
	m := NewQuickPeekModel()
	m.SetSize(120, 24)
	m.SetContent(
		"api-1",
		"Container",
		"Recent Logs",
		"Recent logs · last 128 lines",
		[]string{"status: running"},
		strings.Repeat("line\n", 80),
	)

	before := strings.Split(m.View(), "\n")
	if len(before) != 24 {
		t.Fatalf("expected 24 lines before G, got %d", len(before))
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'G'})

	after := strings.Split(m.View(), "\n")
	if len(after) != 24 {
		t.Fatalf("expected 24 lines after G, got %d", len(after))
	}
}

func TestQuickPeekModel_ReservesBottomLine(t *testing.T) {
	m := NewQuickPeekModel()
	m.SetSize(120, 24)
	m.SetContent(
		"api-1",
		"Container",
		"Recent Logs",
		"Recent logs · last 128 lines",
		[]string{
			"status: running",
			"image: nginx:latest",
			"command: nginx -g daemon off;",
			"ports: 8080->80/tcp",
			"id: abcdef123456",
			"labels: 6",
			"mounts: 2",
			"health: healthy",
		},
		strings.Repeat("line\n", 40),
	)

	v := m.View()
	lines := strings.Split(v, "\n")
	if len(lines) != 24 {
		t.Fatalf("expected 24 lines, got %d", len(lines))
	}
}

// --- FilterInputModel tests ---

func TestFilterInput_ActivateDeactivate(t *testing.T) {
	m := NewFilterInputModel()

	if m.Active() {
		t.Fatal("expected not active initially")
	}

	m.Activate()
	if !m.Active() {
		t.Fatal("expected active after Activate")
	}

	m.Deactivate()
	if m.Active() {
		t.Fatal("expected not active after Deactivate")
	}
}

func TestFilterInput_Clear(t *testing.T) {
	m := NewFilterInputModel()
	m.Activate()
	m.Clear()

	if m.Active() {
		t.Fatal("expected not active after Clear")
	}
	if m.Value() != "" {
		t.Fatalf("expected empty value after Clear, got %q", m.Value())
	}
}

func TestFilterInput_EscClears(t *testing.T) {
	m := NewFilterInputModel()
	m.Activate()

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.Active() {
		t.Fatal("expected not active after esc")
	}
}

func TestFilterInput_EnterConfirms(t *testing.T) {
	m := NewFilterInputModel()
	m.Activate()

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.Active() {
		t.Fatal("expected not active after enter")
	}
}

func TestFilterInput_InactiveIgnoresKeys(t *testing.T) {
	m := NewFilterInputModel()
	// Not active — should ignore all keys
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'a'})
	if cmd != nil {
		t.Fatal("expected nil cmd from inactive filter")
	}
}

func TestFilterInput_ViewEmpty(t *testing.T) {
	m := NewFilterInputModel()
	if m.View() != "" {
		t.Fatal("expected empty view when inactive")
	}
}

func TestFilterInput_ViewNotEmpty(t *testing.T) {
	m := NewFilterInputModel()
	m.SetWidth(80)
	m.Activate()
	if m.View() == "" {
		t.Fatal("expected non-empty view when active")
	}
}

// --- InputPromptModel tests ---

func TestInputPrompt_EscCancels(t *testing.T) {
	m, _ := NewInputPromptModel("Scale:", "3", "service-scale", "svc123")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cmd from esc")
	}
	msg := cmd()
	pr, ok := msg.(InputPromptResultMsg)
	if !ok {
		t.Fatalf("expected InputPromptResultMsg, got %T", msg)
	}
	if !pr.Cancelled {
		t.Fatal("expected cancelled")
	}
	if pr.Tag != "service-scale" || pr.ID != "svc123" {
		t.Fatalf("wrong tag/id: %q/%q", pr.Tag, pr.ID)
	}
}

func TestInputPrompt_View(t *testing.T) {
	m, _ := NewInputPromptModel("Scale:", "3", "service-scale", "svc123")
	m.SetWidth(80)
	v := m.View()
	if v == "" {
		t.Fatal("View() should not be empty")
	}
}

// --- CommandPaletteModel tests ---

func TestCommandPaletteModel_EnterSelectsFirstAction(t *testing.T) {
	m, _ := NewCommandPaletteModel([]CommandPaletteItem{
		{ID: "switch:images", Title: "Go to images"},
		{ID: "global:help", Title: "Open help"},
	})

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd from enter")
	}
	msg := cmd()
	result, ok := msg.(CommandPaletteResultMsg)
	if !ok {
		t.Fatalf("expected CommandPaletteResultMsg, got %T", msg)
	}
	if result.ActionID != "switch:images" {
		t.Fatalf("expected first action id, got %q", result.ActionID)
	}
}

func TestCommandPaletteModel_FilterNarrowsResults(t *testing.T) {
	m, _ := NewCommandPaletteModel([]CommandPaletteItem{
		{ID: "container:logs", Group: "Container", Title: "Logs", Search: "logs output"},
		{ID: "switch:images", Group: "Go To", Title: "Images", Search: "switch images"},
	})
	m.input.SetValue("logs")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected one filtered item, got %d", len(m.filtered))
	}
	if m.filtered[0].ID != "container:logs" {
		t.Fatalf("expected logs action, got %q", m.filtered[0].ID)
	}
}

func TestCommandPaletteModel_FuzzyRankingPrefersPrefixMatches(t *testing.T) {
	m, _ := NewCommandPaletteModel([]CommandPaletteItem{
		{ID: "global:prune", Group: "Docker", Title: "Prune Unused Resources", Search: "cleanup prune"},
		{ID: "switch:compose-projects", Group: "Go To", Title: "Compose Projects", Search: "switch compose"},
	})
	m.input.SetValue("prun")
	m.applyFilter()

	if len(m.filtered) == 0 {
		t.Fatal("expected fuzzy-ranked results")
	}
	if m.filtered[0].ID != "global:prune" {
		t.Fatalf("expected prune action first, got %q", m.filtered[0].ID)
	}
}

func TestCommandPaletteModel_EscCloses(t *testing.T) {
	m, _ := NewCommandPaletteModel([]CommandPaletteItem{{ID: "global:help", Title: "Open help"}})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cmd from esc")
	}
	msg := cmd()
	if _, ok := msg.(CloseOverlayMsg); !ok {
		t.Fatalf("expected CloseOverlayMsg, got %T", msg)
	}
}

func TestCommandPaletteModel_ViewShowsGroupAndFitsNarrowWidth(t *testing.T) {
	m, _ := NewCommandPaletteModel([]CommandPaletteItem{
		{ID: "container:logs", Group: "Container", Title: "Logs", Description: "api-1"},
	})
	m.SetSize(28, 12)

	v := m.View()
	if v == "" {
		t.Fatal("expected non-empty palette view")
	}
	if !strings.Contains(v, "Container") {
		t.Fatal("expected group label in palette view")
	}
}
