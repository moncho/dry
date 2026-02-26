package appui

import (
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
