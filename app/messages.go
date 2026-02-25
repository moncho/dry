package app

import (
	"time"

	"github.com/docker/docker/api/types/events"
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

// Operation result messages

type operationSuccessMsg struct {
	message string
}

type operationErrorMsg struct {
	err error
}

// Internal messages

type refreshMsg struct{}

type statusMessageMsg struct {
	text   string
	expiry time.Duration
}

type flushRefreshMsg struct{}

// View lifecycle messages

type viewActivatedMsg struct {
	view viewMode
}

type viewDeactivatedMsg struct {
	view viewMode
}

// Overlay messages

type showLessMsg struct {
	content string
	title   string
}

type showPromptMsg struct {
	message  string
	callback func(string)
}

// Loading animation message
type loadingTickMsg struct{}
