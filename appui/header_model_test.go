package appui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/moncho/dry/mocks"
)

// At narrow widths the three header columns must not run into each other.
// Regression test for the collision where a long value butted directly
// against the next column's label (e.g. "unix:///var/rDocker Version").
func TestHeaderModel_NarrowWidthKeepsColumnGap(t *testing.T) {
	InitStyles()
	daemon := &mocks.DockerDaemonMock{}

	for _, width := range []int{40, 50, 70} {
		m := NewHeaderModel(daemon, width)
		lines := strings.Split(m.View(), "\n")
		if len(lines) < 3 {
			t.Fatalf("width %d: expected 3 header lines, got %d", width, len(lines))
		}
		for _, label := range []string{"Docker Version:", "Hostname:"} {
			line := ansi.Strip(lines[0])
			idx := strings.Index(line, label)
			if idx <= 0 {
				continue // label may be truncated away at very narrow widths
			}
			if line[idx-1] != ' ' {
				t.Errorf("width %d: %q is not preceded by a space (column collision): %q",
					width, label, line)
			}
		}
	}
}

// Truncation must never erase a value entirely: at narrow widths at least
// one character of the value survives after the label.
func TestHeaderModel_NarrowWidthKeepsValueVisible(t *testing.T) {
	InitStyles()
	daemon := &mocks.DockerDaemonMock{}

	// The mock reports DockerHost "dry.io".
	for _, width := range []int{40, 50, 70} {
		m := NewHeaderModel(daemon, width)
		line := ansi.Strip(strings.Split(m.View(), "\n")[0])
		if !strings.Contains(line, "Docker Host: d") {
			t.Errorf("width %d: expected at least one host value character, got %q", width, line)
		}
	}
}

// Every rendered header line must fit exactly within the configured width.
func TestHeaderModel_LinesFitWidth(t *testing.T) {
	InitStyles()
	daemon := &mocks.DockerDaemonMock{}
	const width = 60
	m := NewHeaderModel(daemon, width)
	for i, line := range strings.Split(m.View(), "\n") {
		if got := ansi.StringWidth(line); got > width {
			t.Errorf("header line %d width = %d, want <= %d: %q", i, got, width, ansi.Strip(line))
		}
	}
}
