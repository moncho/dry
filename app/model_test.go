package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/mocks"
)

func newTestModel() model {
	m := NewModel(Config{})
	m.width = 120
	m.height = 40
	m.daemon = &mocks.DockerDaemonMock{}
	m.ready = true
	m.swarmMode = true
	m.containers.SetDaemon(m.daemon)
	m.images.SetDaemon(m.daemon)
	m.networks.SetDaemon(m.daemon)
	m.volumes.SetDaemon(m.daemon)
	ch := m.contentHeight()
	m.containers.SetSize(m.width, ch)
	m.images.SetSize(m.width, ch)
	m.networks.SetSize(m.width, ch)
	m.volumes.SetSize(m.width, ch)
	return m
}

func TestModel_ViewSwitching(t *testing.T) {
	m := newTestModel()

	tests := []struct {
		key      string
		expected viewMode
	}{
		{"2", Images},
		{"3", Networks},
		{"4", Volumes},
		{"1", Main},
		{"5", Nodes},
		{"6", Services},
		{"7", Stacks},
	}

	for _, tt := range tests {
		result, _ := m.Update(tea.KeyPressMsg{Code: rune(tt.key[0])})
		m = result.(model)
		if m.view != tt.expected {
			t.Errorf("key %q: expected view %d, got %d", tt.key, tt.expected, m.view)
		}
	}
}

func TestModel_HelpOverlay(t *testing.T) {
	m := newTestModel()

	// Press ? to open help
	result, cmd := m.Update(tea.KeyPressMsg{Code: '?'})
	m = result.(model)

	if cmd == nil {
		t.Fatal("expected a cmd from help key")
	}

	// Execute the cmd — it returns showLessMsg
	msg := cmd()
	if msg == nil {
		t.Fatal("expected non-nil msg from help cmd")
	}

	result, _ = m.Update(msg)
	m = result.(model)

	if m.overlay != overlayLess {
		t.Fatalf("expected overlayLess, got %d", m.overlay)
	}
}

func TestModel_CloseOverlay(t *testing.T) {
	m := newTestModel()
	m.overlay = overlayLess

	result, _ := m.Update(appui.CloseOverlayMsg{})
	m = result.(model)

	if m.overlay != overlayNone {
		t.Fatalf("expected overlayNone after CloseOverlayMsg, got %d", m.overlay)
	}
}

func TestModel_PromptConfirm(t *testing.T) {
	m := newTestModel()

	// Show a prompt
	m = m.showPrompt("Test?", "kill", "abc123def456")
	if m.overlay != overlayPrompt {
		t.Fatal("expected overlayPrompt")
	}

	// Confirm — sends PromptResultMsg
	result, cmd := m.Update(tea.KeyPressMsg{Code: 'y'})
	m = result.(model)

	if cmd == nil {
		t.Fatal("expected cmd from prompt confirm")
	}

	// Execute the cmd — should produce PromptResultMsg
	msg := cmd()
	if _, ok := msg.(appui.PromptResultMsg); !ok {
		t.Fatalf("expected PromptResultMsg, got %T", msg)
	}
}

func TestModel_PromptDeny(t *testing.T) {
	m := newTestModel()

	m = m.showPrompt("Test?", "kill", "abc123def456")

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'n'})
	_ = result.(model)

	if cmd == nil {
		t.Fatal("expected cmd from prompt deny")
	}

	msg := cmd()
	pr, ok := msg.(appui.PromptResultMsg)
	if !ok {
		t.Fatalf("expected PromptResultMsg, got %T", msg)
	}
	if pr.Confirmed {
		t.Fatal("expected not confirmed")
	}
}

func TestModel_EscapeGoesToMain(t *testing.T) {
	m := newTestModel()

	// Switch to images
	result, _ := m.Update(tea.KeyPressMsg{Code: '2'})
	m = result.(model)
	if m.view != Images {
		t.Fatalf("expected Images view, got %d", m.view)
	}

	// Press escape — should go to Main
	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = result.(model)
	if m.view != Main {
		t.Fatalf("expected Main view after escape, got %d", m.view)
	}
}

func TestModel_EscapeNoopOnMain(t *testing.T) {
	m := newTestModel()

	if m.view != Main {
		t.Fatalf("expected initial view Main, got %d", m.view)
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = result.(model)
	if m.view != Main {
		t.Fatalf("expected view still Main after escape, got %d", m.view)
	}
}

func TestModel_ContainerMenuOverlay(t *testing.T) {
	m := newTestModel()
	// Load some containers
	containers := m.daemon.Containers(nil, 0)
	m.containers.SetContainers(containers)

	// Press enter — should open container menu
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = result.(model)

	if m.overlay != overlayContainerMenu {
		t.Fatalf("expected overlayContainerMenu, got %d", m.overlay)
	}
}

func TestModel_F7TogglesHeader(t *testing.T) {
	m := newTestModel()

	initialHeader := m.showHeader
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	m = result.(model)

	if m.showHeader == initialHeader {
		t.Fatal("expected header toggle")
	}

	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	m = result.(model)

	if m.showHeader != initialHeader {
		t.Fatal("expected header toggled back")
	}
}

func TestModel_ContainersLoadedMsg(t *testing.T) {
	m := newTestModel()
	containers := m.daemon.Containers(nil, 0)

	result, _ := m.Update(containersLoadedMsg{containers: containers})
	m = result.(model)

	if m.containers.SelectedContainer() == nil {
		t.Fatal("expected containers to be loaded and selectable")
	}
}

func TestModel_OperationSuccessMsg(t *testing.T) {
	m := newTestModel()

	result, cmd := m.Update(operationSuccessMsg{message: "done!"})
	_ = result.(model)

	// Should trigger a reload for the current view
	if cmd == nil {
		t.Fatal("expected cmd from operationSuccessMsg")
	}
}

func TestModel_StatusMessageMsg(t *testing.T) {
	m := newTestModel()

	result, _ := m.Update(statusMessageMsg{text: "test message"})
	_ = result.(model)
	// No crash is sufficient — message bar state is internal
}

func TestModel_ShortID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc123def456789", "abc123def456"},
		{"short", "short"},
		{"exactly12ch", "exactly12ch"},
		{"", ""},
	}

	for _, tt := range tests {
		got := shortID(tt.input)
		if got != tt.expected {
			t.Errorf("shortID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestModel_LessScrolling(t *testing.T) {
	m := newTestModel()

	// Create a showLessMsg with many lines
	lines := make([]string, 100)
	for i := range 100 {
		lines[i] = fmt.Sprintf("Line %d: content", i+1)
	}
	content := strings.Join(lines, "\n")

	result, _ := m.Update(showLessMsg{content: content, title: "Test"})
	m = result.(model)

	if m.overlay != overlayLess {
		t.Fatalf("expected overlayLess, got %d", m.overlay)
	}

	v1 := m.View()

	// Scroll down with 'j'
	result, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m = result.(model)
	result, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m = result.(model)
	result, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m = result.(model)

	v2 := m.View()

	if v1.Content == v2.Content {
		t.Fatal("expected view to change after scrolling in less overlay")
	}
}

func TestModel_ResizeWithOverlay(t *testing.T) {
	m := newTestModel()

	// Open less overlay
	lines := make([]string, 50)
	for i := range 50 {
		lines[i] = fmt.Sprintf("Line %d", i+1)
	}
	result, _ := m.Update(showLessMsg{content: strings.Join(lines, "\n"), title: "Test"})
	m = result.(model)

	if m.overlay != overlayLess {
		t.Fatalf("expected overlayLess, got %d", m.overlay)
	}

	v1 := m.View()

	// Resize terminal
	result, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = result.(model)

	if m.width != 60 || m.height != 20 {
		t.Fatalf("expected 60x20, got %dx%d", m.width, m.height)
	}

	v2 := m.View()
	if v1.Content == v2.Content {
		t.Fatal("expected view to change after resize with less overlay active")
	}
}
