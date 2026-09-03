package app

import (
	"io"
	"time"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/composecli"
)

// Docker data messages

type containersLoadedMsg struct {
	containers []*docker.Container
}

type dockerConnectedMsg struct {
	daemon dockerDaemon
}

type dockerErrorMsg struct {
	err error
}

type dockerEventMsg struct {
	event events.Message
}

type eventsClosedMsg struct{}

type reconnectEventsMsg struct{}

// composeDetectedMsg carries the result of probing for the compose plugin.
type composeDetectedMsg struct {
	cli *composecli.CLI
	err error
}

// composeProjectsMsg carries a project list enriched with a compose file
// discovered on disk. gen is the cycle's generation, carried through to
// the drift check the model dispatches next.
type composeProjectsMsg struct {
	projects []docker.ProjectWithServices
	gen      uint64
}

// composeDriftMsg carries per-project, per-service sync status, plus why any
// project's check did not complete. composeDriftState.merge decides what it
// is allowed to change; see there for the generation and failure rules.
type composeDriftMsg struct {
	// project names the single project this message covers, empty for a whole
	// cycle over the list. The model merges the former and replaces on the
	// latter, so one project's SYNC does not blank the rest.
	project  string
	gen      uint64
	drift    map[string]map[string]docker.ServiceSync
	failures map[string]string
}

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

// headerInfoMsg carries the asynchronously fetched daemon info and version
// for the header.
type headerInfoMsg struct {
	info    system.Info
	infoErr error
	ver     *client.ServerVersionResult
	verErr  error
}

// streamClosedMsg signals the streaming reader has ended.
type streamClosedMsg struct {
	reader io.ReadCloser // the reader that ended, so a stale close is ignorable
	err    error         // the process's exit error, if Close reported one
}

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
