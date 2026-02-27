package app

import (
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
