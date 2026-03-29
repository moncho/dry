package app

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/moncho/dry/appui"
)

func TestCollectQuickPeekLogContentReadsAdditionalChunks(t *testing.T) {
	pr, pw := io.Pipe()
	go func() {
		_, _ = pw.Write([]byte("line2\n"))
		time.Sleep(10 * time.Millisecond)
		_, _ = pw.Write([]byte("line3\n"))
		_ = pw.Close()
	}()

	content, err := collectQuickPeekLogContent(streamingContent{
		content: "line1\n",
		reader:  pr,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if want := "line1\nline2\nline3\n"; content != want {
		t.Fatalf("expected %q, got %q", want, content)
	}
	if strings.Count(content, "\n") != 3 {
		t.Fatalf("expected three log lines, got %q", content)
	}
}

func TestWorkspaceMonitorDetailContentShowsGaugeSections(t *testing.T) {
	content := workspaceMonitorDetailContent(workspaceContext{
		title:      "redis",
		subtitle:   "Monitor",
		lines:      []string{"name: redis", "ports: 6379->6379/tcp", "net io: 2kB / 4kB"},
		monitorCPU: 37.5,
		monitorMem: 128 * 1024 * 1024,
		monitorMax: 1024 * 1024 * 1024,
		monitorPct: 12.5,
		monitorCPUHistory: []appui.MonitorPoint{
			{At: time.Unix(0, 0), Value: 5},
			{At: time.Unix(1, 0), Value: 10},
			{At: time.Unix(2, 0), Value: 22},
			{At: time.Unix(3, 0), Value: 37.5},
		},
		monitorMemHistory: []appui.MonitorPoint{
			{At: time.Unix(0, 0), Value: 3},
			{At: time.Unix(1, 0), Value: 6},
			{At: time.Unix(2, 0), Value: 9},
			{At: time.Unix(3, 0), Value: 12.5},
		},
	}, 96, 18)

	plain := ansi.Strip(content)
	for _, want := range []string{"Legend:", "CPU arc", "Memory line", "Y auto-scaled", "CPU", "Memory", "now  37.5%", "128MiB", "12.5%"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("expected %q in monitor detail content, got %q", want, plain)
		}
	}
	for _, unwanted := range []string{"name: redis", "ports: 6379->6379/tcp", "Monitor", "3s/3m"} {
		if strings.Contains(plain, unwanted) {
			t.Fatalf("did not expect %q in monitor detail content, got %q", unwanted, plain)
		}
	}
}

func TestLoadWorkspaceMonitorDetailsStatusShowsCollectedDuration(t *testing.T) {
	msg := loadWorkspaceMonitorDetailsFromContext(workspaceContext{
		title:      "redis",
		monitorCPU: 37.5,
		monitorCPUHistory: []appui.MonitorPoint{
			{At: time.Unix(0, 0), Value: 5},
			{At: time.Unix(16, 0), Value: 37.5},
		},
		monitorMem: 128 * 1024 * 1024,
		monitorMax: 1024 * 1024 * 1024,
		monitorPct: 12.5,
		monitorMemHistory: []appui.MonitorPoint{
			{At: time.Unix(0, 0), Value: 3},
			{At: time.Unix(16, 0), Value: 12.5},
		},
	}, 96, 18)().(workspaceActivityLoadedMsg)

	if got, want := msg.status, "Live stats · 16s/3m window"; got != want {
		t.Fatalf("expected status %q, got %q", want, got)
	}
}

func TestMonitorChartWidthUsesAvailableActivityWidth(t *testing.T) {
	if got, want := monitorChartWidth(60), 56; got != want {
		t.Fatalf("expected wide half-width chart width %d, got %d", want, got)
	}
	if got, want := monitorChartWidth(36), 32; got != want {
		t.Fatalf("expected medium half-width chart width %d, got %d", want, got)
	}
	if got, want := monitorChartWidth(10), 20; got != want {
		t.Fatalf("expected minimum chart width %d, got %d", want, got)
	}
}

func TestMonitorChartYRangeTracksObservedValues(t *testing.T) {
	minY, maxY := monitorChartYRange([]appui.MonitorPoint{
		{Value: 41},
		{Value: 43},
		{Value: 42},
	})
	if minY >= 41 || maxY <= 43 {
		t.Fatalf("expected padded dynamic range around samples, got [%v, %v]", minY, maxY)
	}
	if maxY-minY >= 100 {
		t.Fatalf("expected tighter dynamic range than full scale, got [%v, %v]", minY, maxY)
	}

	minY, maxY = monitorChartYRange([]appui.MonitorPoint{
		{Value: 0.1},
		{Value: 0.4},
	})
	if minY < 0 {
		t.Fatalf("expected lower bound to clamp at 0, got %v", minY)
	}
	if maxY <= 0.4 {
		t.Fatalf("expected upper bound above observed values, got %v", maxY)
	}
}

func TestMonitorChartHeightFitsAvailableBody(t *testing.T) {
	// Proportional: chartHeightPct of (bodyHeight - chartOverhead), clamped to [minChartHeight, available].
	if got, want := monitorChartHeight(12), 4; got != want {
		t.Fatalf("expected chart height %d for moderate body, got %d", want, got)
	}
	if got, want := monitorChartHeight(20), 9; got != want {
		t.Fatalf("expected taller chart height %d for roomy pane, got %d", want, got)
	}
	if got, want := monitorChartHeight(7), 3; got != want {
		t.Fatalf("expected minimum chart height %d for tight pane, got %d", want, got)
	}
}

func TestTrimMonitorHistoryKeepsLastThreeMinutes(t *testing.T) {
	now := time.Unix(600, 0)
	samples := []appui.MonitorPoint{
		{At: now.Add(-5 * time.Minute), Value: 10},
		{At: now.Add(-4 * time.Minute), Value: 20},
		{At: now.Add(-2 * time.Minute), Value: 30},
		{At: now.Add(-30 * time.Second), Value: 40},
	}
	trimmed := trimMonitorHistory(samples, 3*time.Minute)
	if got, want := len(trimmed), 2; got != want {
		t.Fatalf("expected %d samples in 3 minute window, got %d", want, got)
	}
	if trimmed[0].Value != 30 || trimmed[1].Value != 40 {
		t.Fatalf("expected only recent samples to remain, got %+v", trimmed)
	}
}

func TestMonitorCollectedDurationCapsAtWindow(t *testing.T) {
	now := time.Unix(600, 0)
	samples := []appui.MonitorPoint{
		{At: now.Add(-5 * time.Minute), Value: 10},
		{At: now, Value: 40},
	}
	if got, want := monitorCollectedDuration(samples, 3*time.Minute), 3*time.Minute; got != want {
		t.Fatalf("expected collected duration %v, got %v", want, got)
	}
	if got, want := formatMonitorDuration(95*time.Second), "1m35s"; got != want {
		t.Fatalf("expected formatted duration %q, got %q", want, got)
	}
}
