package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/mocks"
)

func TestRenderMainScreen_LineCount(t *testing.T) {
	appui.InitStyles()
	m := newTestModel()
	m.header = appui.NewHeaderModel(m.daemon, m.width)

	containers := m.daemon.Containers(nil, docker.SortByContainerID)

	result, _ := m.Update(containersLoadedMsg{containers: containers})
	m = result.(model)

	hdrLines := strings.Split(m.header.View(), "\n")
	sepLines := strings.Split(m.header.SeparatorLine(""), "\n")
	t.Logf("Header lines: %d, Separator lines: %d", len(hdrLines), len(sepLines))

	contentLines := strings.Split(m.containers.View(), "\n")
	t.Logf("Content lines: %d (expected contentHeight=%d)", len(contentLines), m.contentHeight())

	v := m.renderMainScreen()
	lines := strings.Split(v, "\n")
	t.Logf("Terminal: %dx%d, Total lines: %d (expected %d)", m.width, m.height, len(lines), m.height)

	dataRows := 0
	for _, l := range contentLines {
		if strings.Contains(l, "▶") || strings.Contains(l, "■") {
			dataRows++
		}
	}
	t.Logf("Data rows: %d", dataRows)

	if len(lines) != m.height {
		t.Errorf("Expected %d lines, got %d", m.height, len(lines))
	}

	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = result.(model)
	contentLines2 := strings.Split(m.containers.View(), "\n")
	dataRows2 := 0
	for _, l := range contentLines2 {
		if strings.Contains(l, "▶") || strings.Contains(l, "■") {
			dataRows2++
		}
	}
	t.Logf("Data rows after nav: %d", dataRows2)
	if dataRows != dataRows2 {
		t.Errorf("Data row count changed: %d -> %d", dataRows, dataRows2)
	}
}

func TestRenderMainScreen_SmallTerminal(t *testing.T) {
	// Simulate a small terminal where rows exceed viewport capacity
	appui.InitStyles()
	m := NewModel(Config{})
	m.width = 120
	m.height = 25 // small terminal
	m.daemon = &mocks.DockerDaemonMock{}
	m.ready = true
	m.containers.SetDaemon(m.daemon)
	m.header = appui.NewHeaderModel(m.daemon, m.width)

	ch := m.contentHeight()
	m.containers.SetSize(m.width, ch)

	// 20 containers > viewport capacity
	containers := m.daemon.Containers(nil, docker.SortByContainerID)
	result, _ := m.Update(containersLoadedMsg{containers: containers})
	m = result.(model)

	contentLines := strings.Split(m.containers.View(), "\n")
	dataRows := 0
	for _, l := range contentLines {
		if strings.Contains(l, "▶") || strings.Contains(l, "■") {
			dataRows++
		}
	}
	// Table height = contentHeight - 1 (widget header)
	// Visible data = tableHeight - 1 (table header)
	expectedDataRows := ch -
		1 - // widget header
		1 - // blank line
		1 - // table header
		1 // blank line after the table
	if dataRows < len(containers) {
		// More containers than viewport - should show viewport's worth
		t.Logf("Terminal %dx%d, contentHeight=%d, tableH=%d", m.width, m.height, ch, ch-1)
		t.Logf("Visible data rows: %d (expected %d)", dataRows, expectedDataRows)
		if dataRows != expectedDataRows {
			t.Errorf("Expected %d visible data rows, got %d", expectedDataRows, dataRows)
		}
	}

	// Navigate and check
	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = result.(model)
	contentLines2 := strings.Split(m.containers.View(), "\n")
	dataRows2 := 0
	for _, l := range contentLines2 {
		if strings.Contains(l, "▶") || strings.Contains(l, "■") {
			dataRows2++
		}
	}
	t.Logf("Data rows after nav: %d", dataRows2)
	if dataRows != dataRows2 {
		t.Errorf("Data row count changed after nav: %d -> %d", dataRows, dataRows2)
	}

	v := m.renderMainScreen()
	totalLines := len(strings.Split(v, "\n"))
	t.Logf("Total screen lines: %d (expected %d)", totalLines, m.height)
	if totalLines != m.height {
		t.Errorf("Expected %d total lines, got %d", m.height, totalLines)
	}
}

func TestRenderWorkspaceScreen_ContextOverflowDoesNotHideFooter(t *testing.T) {
	appui.InitStyles()
	m := NewModel(Config{WorkspaceMode: true})
	m.width = 120
	m.height = 30
	m.daemon = &mocks.DockerDaemonMock{}
	m.ready = true
	m.containers.SetDaemon(m.daemon)
	m.header = appui.NewHeaderModel(m.daemon, m.width)
	m.resizeContentModels()

	m.pinnedContext = &workspaceContext{
		title:    "overflow",
		subtitle: "Context",
		lines: []string{
			"one: 1", "two: 2", "three: 3", "four: 4", "five: 5",
			"six: 6", "seven: 7", "eight: 8", "nine: 9", "ten: 10",
			"eleven: 11", "twelve: 12", "thirteen: 13", "fourteen: 14",
		},
	}

	v := m.renderMainScreen()
	totalLines := len(strings.Split(v, "\n"))
	if totalLines != m.height {
		t.Fatalf("expected %d total lines, got %d", m.height, totalLines)
	}
}

func TestRenderWorkspaceScreen_LongContextLinesDoNotHideFooter(t *testing.T) {
	appui.InitStyles()
	m := NewModel(Config{WorkspaceMode: true})
	m.width = 120
	m.height = 24
	m.daemon = &mocks.DockerDaemonMock{}
	m.ready = true
	m.containers.SetDaemon(m.daemon)
	m.header = appui.NewHeaderModel(m.daemon, m.width)
	m.resizeContentModels()

	m.pinnedContext = &workspaceContext{
		title:    "wrapped",
		subtitle: "Context",
		lines: []string{
			"labels: this-is-a-very-long-context-value-that-should-be-truncated-before-it-wraps-and-pushes-the-footer-off-screen",
			"mounts: /a/very/long/path/that/keeps/going/and/going/until-it-would-normally-wrap-inside-the-pane",
		},
	}

	v := m.renderMainScreen()
	totalLines := len(strings.Split(v, "\n"))
	if totalLines != m.height {
		t.Fatalf("expected %d total lines, got %d", m.height, totalLines)
	}
	if strings.Contains(v, "pushes-the-footer-off-screen") {
		t.Fatal("expected long context value to be truncated before wrapping")
	}
}

func TestRenderWorkspaceScreen_NarrowTerminalUsesCompactFallback(t *testing.T) {
	appui.InitStyles()
	m := NewModel(Config{WorkspaceMode: true})
	m.width = 90
	m.height = 28
	m.daemon = &mocks.DockerDaemonMock{}
	m.ready = true
	m.containers.SetDaemon(m.daemon)
	m.header = appui.NewHeaderModel(m.daemon, m.width)
	m.resizeContentModels()

	containers := m.daemon.Containers(nil, docker.SortByContainerID)
	result, _ := m.Update(containersLoadedMsg{containers: containers})
	m = result.(model)

	if !m.workspaceCompactMode() {
		t.Fatal("expected narrow workspace screen to use compact fallback")
	}

	v := m.renderMainScreen()
	totalLines := len(strings.Split(v, "\n"))
	if totalLines != m.height {
		t.Fatalf("expected %d total lines, got %d", m.height, totalLines)
	}
	if !strings.Contains(v, "Navigator") || !strings.Contains(v, "Activity") {
		t.Fatal("expected workspace tabs in compact mode")
	}
	if strings.Contains(v, "Context ·") {
		t.Fatal("did not expect context pane to render in compact workspace mode")
	}
}

func TestRenderWorkspaceScreen_ShortTerminalShowsActivityPaneInCompactMode(t *testing.T) {
	appui.InitStyles()
	m := NewModel(Config{WorkspaceMode: true})
	m.width = 120
	m.height = 14
	m.daemon = &mocks.DockerDaemonMock{}
	m.ready = true
	m.header = appui.NewHeaderModel(m.daemon, m.width)
	m.activePane = workspacePaneActivity
	m.resizeContentModels()
	m.workspaceLogs.SetContent("Activity", "Pinned activity", "line one\nline two")

	if !m.workspaceCompactMode() {
		t.Fatal("expected short workspace screen to use compact fallback")
	}

	v := m.renderMainScreen()
	totalLines := len(strings.Split(v, "\n"))
	if totalLines != m.height {
		t.Fatalf("expected %d total lines, got %d", m.height, totalLines)
	}
	if !strings.Contains(v, "Pinned activity") {
		t.Fatal("expected activity pane content in compact mode")
	}
	if strings.Contains(v, "Context ·") {
		t.Fatal("did not expect context pane to render in short compact mode")
	}
}

func TestRenderWorkspaceScreen_MonitorViewDoesNotGrowPastTopPane(t *testing.T) {
	appui.InitStyles()
	m := NewModel(Config{WorkspaceMode: true})
	m.width = 120
	m.height = 30
	m.daemon = &mocks.DockerDaemonMock{}
	m.ready = true
	m.view = Monitor
	m.header = appui.NewHeaderModel(m.daemon, m.width)
	m.resizeContentModels()

	_, _, topH, activityH := m.workspaceLayout()
	if topH != 5 {
		t.Fatalf("expected empty monitor top pane height 5, got %d", topH)
	}
	if activityH <= 0 {
		t.Fatalf("expected positive activity height, got %d", activityH)
	}

	screen := m.renderMainScreen()
	if got := len(strings.Split(screen, "\n")); got != m.height {
		t.Fatalf("expected rendered screen height %d, got %d", m.height, got)
	}

	body := m.renderWorkspaceBody()
	bodyLines := strings.Split(body, "\n")
	if len(bodyLines) != m.contentHeight() {
		t.Fatalf("expected workspace body to fill %d lines, got %d", m.contentHeight(), len(bodyLines))
	}
}

func TestRenderWorkspaceScreen_MonitorFooterVisibleAfterStatsArrive(t *testing.T) {
	appui.InitStyles()
	m := NewModel(Config{WorkspaceMode: true})
	m.width = 120
	m.height = 30
	m.daemon = &mocks.DockerDaemonMock{}
	m.ready = true
	m.view = Monitor
	m.header = appui.NewHeaderModel(m.daemon, m.width)
	m.resizeContentModels()

	// Simulate stats arriving for several containers so RowCount changes.
	ch := make(chan *docker.Stats)
	for i := 0; i < 5; i++ {
		cid := fmt.Sprintf("cid%02d", i)
		msg := appui.MonitorStatsMsg{
			CID:     cid,
			Stats:   &docker.Stats{CID: cid, CPUPercentage: float64(i * 10)},
			StatsCh: ch,
		}
		updated, _ := m.Update(msg)
		m = updated.(model)
	}

	if m.monitor.RowCount() != 5 {
		t.Fatalf("expected 5 monitor rows, got %d", m.monitor.RowCount())
	}

	screen := m.renderMainScreen()
	if got := len(strings.Split(screen, "\n")); got != m.height {
		t.Fatalf("expected rendered screen height %d, got %d", m.height, got)
	}
}
