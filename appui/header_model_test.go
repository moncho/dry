package appui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/moncho/dry/mocks"
)

// newLoadedHeader builds a header with the mock daemon's info already
// delivered, the way the headerInfoMsg handler does in production.
func newLoadedHeader(daemon *mocks.DockerDaemonMock, width int) HeaderModel {
	m := NewHeaderModel(daemon, width)
	info, infoErr := daemon.Info()
	ver, verErr := daemon.Version()
	m.SetDockerInfo(info, infoErr, ver, verErr)
	return m
}

// Until SetDockerInfo delivers the daemon info, the header renders a
// placeholder of the same height so the layout does not jump, and never
// flashes an error for data that simply has not arrived yet.
func TestHeaderModel_PlaceholderBeforeInfoLoads(t *testing.T) {
	InitStyles()
	m := NewHeaderModel(&mocks.DockerDaemonMock{}, 60)

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected the placeholder to keep the 3-line header height, got %d lines", len(lines))
	}
	if strings.Contains(view, "Error") {
		t.Fatalf("expected no error while info is loading, got %q", view)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got != 60 {
			t.Errorf("placeholder line %d width = %d, want 60", i, got)
		}
	}
}

// At narrow widths the three header columns must not run into each other.
// Regression test for the collision where a long value butted directly
// against the next column's label (e.g. "unix:///var/rDocker Version").
func TestHeaderModel_NarrowWidthKeepsColumnGap(t *testing.T) {
	InitStyles()
	daemon := &mocks.DockerDaemonMock{}

	for _, width := range []int{40, 50, 70} {
		m := newLoadedHeader(daemon, width)
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
		m := newLoadedHeader(daemon, width)
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
	m := newLoadedHeader(daemon, width)
	for i, line := range strings.Split(m.View(), "\n") {
		if got := ansi.StringWidth(line); got > width {
			t.Errorf("header line %d width = %d, want <= %d: %q", i, got, width, ansi.Strip(line))
		}
	}
}

// The separator is one line in the layout's height budget, so anything the
// message bar carries has to end up on one line: errors.Join produces
// newline-separated text (a compose drift check reports one failure per
// project), and a message wider than the terminal would be wrapped by
// lipgloss. Either way the extra lines push the footer off the screen.
func TestHeaderModel_SeparatorLineKeepsMessagesToOneLine(t *testing.T) {
	InitStyles()
	m := newLoadedHeader(&mocks.DockerDaemonMock{}, 40)

	for name, message := range map[string]string{
		"joined errors":   "web: config timed out\napi: config timed out\ndb: config timed out",
		"overlong":        strings.Repeat("compose drift check failed ", 10),
		"both":            strings.Repeat("a", 60) + "\n" + strings.Repeat("b", 60),
		"carriage return": "web: pulling\rweb: pulled",
		"backspace":       "web: 50%\b\b\b100%",
		"bell":            "web: done\a",
		"tab":             "web:\tconfig timed out",
	} {
		t.Run(name, func(t *testing.T) {
			line := m.SeparatorLine(message)
			if got := strings.Count(line, "\n"); got != 0 {
				t.Fatalf("expected one line, got %d newlines in %q", got, ansi.Strip(line))
			}
			if got := ansi.StringWidth(line); got != 40 {
				t.Fatalf("expected the separator to be exactly 40 wide, got %d", got)
			}
		})
	}

	// Width and newline count do not see a control character: lipgloss
	// passes \r, \b and BEL through untouched and counts them as zero
	// width, so only the text says whether they were removed.
	for name, tc := range map[string]struct{ in, want string }{
		"carriage return": {"web: pulling\rweb: pulled", "web: pullingweb: pulled"},
		"backspace":       {"web: 50%\b\b\b100%", "web: 50%100%"},
		"bell":            {"web: done\a", "web: done"},
		"tab":             {"web:\tconfig timed out", "web: config timed out"},
		"newline":         {"web: one\napi: two", "web: one; api: two"},
	} {
		t.Run("text/"+name, func(t *testing.T) {
			got := strings.TrimRight(ansi.Strip(m.SeparatorLine(tc.in)), " ")
			if got != tc.want {
				t.Errorf("SeparatorLine(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

	// The first failure still has to be readable, not replaced by an
	// ellipsis at the front.
	line := ansi.Strip(m.SeparatorLine("web: config timed out\napi: config timed out"))
	if !strings.HasPrefix(line, "web: config timed out") {
		t.Fatalf("expected the first message to survive, got %q", line)
	}
}
