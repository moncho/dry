package app

import (
	"io"
	"time"

	"github.com/moby/moby/api/types/events"
	"github.com/moncho/dry/docker"
)

// Docker data messages

type containersLoadedMsg struct {
	containers []*docker.Container
}

type dockerConnectedMsg struct {
	daemon docker.ContainerDaemon
}

type dockerErrorMsg struct {
	err error
}

type dockerEventMsg struct {
	event events.Message
}

type eventsClosedMsg struct{}

type reconnectEventsMsg struct{}

// Operation result messages

type operationSuccessMsg struct {
	message string
}

type operationErrorMsg struct {
	err error
}

// Internal messages

type statusMessageMsg struct {
	text   string
	expiry time.Duration
}

type flushRefreshMsg struct{}

type flushMonitorStatsMsg struct{}

// messageBarExpiredMsg triggers a re-render so the expired message clears.
type messageBarExpiredMsg struct{}

// Overlay messages

type showLessMsg struct {
	content string
	title   string
}

// showStreamingLessMsg opens a less viewer with initial content and a
// reader that will be streamed via appendLessMsg.
type showStreamingLessMsg struct {
	content string
	title   string
	reader  io.ReadCloser
}

// appendLessMsg appends streamed content to an open less viewer.
type appendLessMsg struct {
	content string
	reader  io.ReadCloser // passed back for the next read cycle
}

// streamClosedMsg signals the streaming reader has ended.
type streamClosedMsg struct{}

type workspaceActivityLoadedMsg struct {
	title   string
	status  string
	content string
	reader  io.ReadCloser
}

type appendWorkspaceActivityMsg struct {
	content string
	reader  io.ReadCloser
}

type workspaceActivityClosedMsg struct{}

type quickPeekLoadedMsg struct {
	title       string
	subtitle    string
	detailTitle string
	status      string
	summary     []string
	content     string
}

// Loading animation message
type loadingTickMsg struct{}

// splashDoneMsg signals the splash timer has elapsed.
type splashDoneMsg struct{}

// execEndedMsg signals that a tea.Exec session has completed.
// It carries a status message and triggers a screen repaint.
type execEndedMsg struct {
	text   string
	expiry time.Duration
}
