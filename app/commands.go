package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	timeserieslinechart "github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-units"
	"github.com/moncho/dry/appui"
	appcompose "github.com/moncho/dry/appui/compose"
	appswarm "github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"golang.org/x/term"
)

const monitorChartWindow = 3 * time.Minute

// Monitor chart layout constants.
const (
	chartGap       = 4  // horizontal gap between side-by-side charts
	chartPadding   = 4  // horizontal padding within each chart panel
	minChartWidth  = 20 // minimum chart render width
	maxChartWidth  = 56 // maximum chart render width
	minChartHeight = 3  // minimum chart render height
	chartOverhead  = 4  // non-chart lines: legend + blank + title + detail
	chartHeightPct = 60 // percentage of available body height used for chart
)

// demuxedReadCloser wraps a pipe reader and closes the underlying Docker stream.
type demuxedReadCloser struct {
	io.Reader
	close func() error
}

type streamingContent struct {
	title   string
	content string
	reader  io.ReadCloser
}

type logReadResult struct {
	content string
	err     error
	eof     bool
}

const (
	workspaceActivityLogTail = 512
	quickPeekLogTail         = 128
	quickPeekReadIdle        = 150 * time.Millisecond
	quickPeekReadMax         = 750 * time.Millisecond
)

func (d *demuxedReadCloser) Close() error {
	return d.close()
}

// demuxDockerStream creates a pipe that demultiplexes Docker's multiplexed
// log stream (stdout/stderr interleaved with 8-byte headers) into clean text.
func demuxDockerStream(raw io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()
	go func() {
		_, err := stdcopy.StdCopy(pw, pw, raw)
		if err != nil {
			pw.CloseWithError(err)
		} else {
			_ = pw.Close()
		}
		_ = raw.Close()
	}()
	return &demuxedReadCloser{
		Reader: pr,
		close: func() error {
			// Close the pipe reader, which causes StdCopy to get a write
			// error on the pipe writer, ending the goroutine.
			return pr.Close()
		},
	}
}

// readLogStreamCmd reads the next chunk from a streaming reader.
func readLogStreamCmd(reader io.ReadCloser) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 32*1024)
		n, err := reader.Read(buf)
		if n > 0 {
			return appendLessMsg{
				content: string(buf[:n]),
				reader:  reader,
			}
		}
		if err != nil {
			_ = reader.Close()
			return streamClosedMsg{}
		}
		// n == 0 && err == nil: valid per io.Reader contract, retry after brief pause.
		time.Sleep(10 * time.Millisecond)
		return readLogStreamCmd(reader)()
	}
}

func readWorkspaceActivityCmd(reader io.ReadCloser) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 32*1024)
		n, err := reader.Read(buf)
		if n > 0 {
			return appendWorkspaceActivityMsg{
				content: string(buf[:n]),
				reader:  reader,
			}
		}
		if err != nil {
			_ = reader.Close()
			return workspaceActivityClosedMsg{}
		}
		// n == 0 && err == nil: valid per io.Reader contract, retry after brief pause.
		time.Sleep(10 * time.Millisecond)
		return readWorkspaceActivityCmd(reader)()
	}
}

func collectQuickPeekLogContent(stream streamingContent) (string, error) {
	content := stream.content
	if stream.reader == nil {
		return content, nil
	}
	defer func() {
		_ = stream.reader.Close()
	}()

	results := make(chan logReadResult, 1)
	readOnce := func() {
		go func() {
			buf := make([]byte, 32*1024)
			n, err := stream.reader.Read(buf)
			switch {
			case n > 0:
				results <- logReadResult{
					content: string(buf[:n]),
					err:     err,
					eof:     errors.Is(err, io.EOF),
				}
			case err != nil:
				results <- logReadResult{
					err: err,
					eof: errors.Is(err, io.EOF),
				}
			default:
				results <- logReadResult{}
			}
		}()
	}

	// drainReadGoroutine closes the reader to unblock a pending Read,
	// then drains the goroutine's result so it can exit.
	drainReadGoroutine := func() {
		_ = stream.reader.Close()
		<-results
	}

	readOnce()
	idle := time.NewTimer(quickPeekReadIdle)
	defer idle.Stop()
	maxWait := time.NewTimer(quickPeekReadMax)
	defer maxWait.Stop()

	for {
		select {
		case result := <-results:
			if result.content != "" {
				content += result.content
			}
			if result.err != nil && !result.eof {
				return content, result.err
			}
			if result.eof {
				return content, nil
			}
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(quickPeekReadIdle)
			readOnce()
		case <-idle.C:
			drainReadGoroutine()
			return content, nil
		case <-maxWait.C:
			drainReadGoroutine()
			return content, nil
		}
	}
}

// shortID safely truncates an ID to at most 12 characters.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func loadContainerLogStream(daemon docker.ContainerDaemon, id string) (streamingContent, error) {
	return loadContainerLogStreamWithTail(daemon, id, workspaceActivityLogTail)
}

func loadContainerLogStreamWithTail(daemon docker.ContainerDaemon, id string, tail int) (streamingContent, error) {
	reader, err := daemon.Logs(id, fmt.Sprintf("tail:%d", tail), false)
	if err != nil {
		return streamingContent{}, err
	}
	if reader == nil {
		return streamingContent{}, errors.New("log stream unavailable")
	}
	return streamingContent{
		title:   fmt.Sprintf("Logs: %s", shortID(id)),
		reader:  demuxDockerStream(reader),
	}, nil
}

func loadComposeLogStream(daemon docker.ContainerDaemon, project, service string) (streamingContent, error) {
	return loadComposeLogStreamWithTail(daemon, project, service, workspaceActivityLogTail)
}

func loadComposeLogStreamWithTail(daemon docker.ContainerDaemon, project, service string, tail int) (streamingContent, error) {
	containers := daemon.Containers(nil, docker.NoSort)
	var targets []*docker.Container
	for _, c := range containers {
		if c.Labels["com.docker.compose.project"] != project {
			continue
		}
		if service != "" && c.Labels["com.docker.compose.service"] != service {
			continue
		}
		if !docker.IsContainerRunning(c) {
			continue
		}
		targets = append(targets, c)
	}
	if len(targets) == 0 {
		return streamingContent{}, errors.New("no running containers found")
	}

	svcCount := make(map[string]int)
	for _, c := range targets {
		svcCount[c.Labels["com.docker.compose.service"]]++
	}

	named := make(map[string]io.ReadCloser)
	for _, c := range targets {
		svcName := c.Labels["com.docker.compose.service"]
		reader, err := daemon.Logs(c.ID, fmt.Sprintf("tail:%d", tail), false)
		if err != nil || reader == nil {
			continue
		}
		key := svcName
		if svcCount[svcName] > 1 {
			key = svcName + "/" + shortID(c.ID)
		}
		named[key] = demuxDockerStream(reader)
	}
	if len(named) == 0 {
		return streamingContent{}, errors.New("could not open any log streams")
	}

	title := fmt.Sprintf("Logs: %s", project)
	if service != "" {
		title = fmt.Sprintf("Logs: %s/%s", project, service)
	}
	return streamingContent{
		title:   title,
		reader:  mergeLogReaders(named),
	}, nil
}

// interactiveExecCommand implements tea.ExecCommand for attach and exec sessions.
type interactiveExecCommand struct {
	daemon     docker.ContainerDaemon
	id         string
	command    []string // nil for attach, non-nil for exec
	waitForKey bool     // pause for keypress after command finishes
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
}

func (c *interactiveExecCommand) SetStdin(r io.Reader)  { c.stdin = r }
func (c *interactiveExecCommand) SetStdout(w io.Writer) { c.stdout = w }
func (c *interactiveExecCommand) SetStderr(w io.Writer) { c.stderr = w }

func (c *interactiveExecCommand) Run() error {
	// Bubbletea passes /dev/tty (not os.Stdin) as the input reader.
	// Use its fd for raw mode so keystrokes are sent immediately.
	var fd int
	if f, ok := c.stdin.(*os.File); ok {
		fd = int(f.Fd())
	} else {
		fd = int(os.Stdin.Fd())
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set terminal raw mode: %w", err)
	}

	var runErr error
	if c.command != nil {
		// When waitForKey is true, don't forward stdin to the exec
		// session so it remains available for the "press any key" read.
		runErr = c.daemon.ExecInteractive(
			context.Background(),
			c.id,
			c.command,
			c.stdin,
			c.stdout,
			c.stderr,
			!c.waitForKey,
		)
	} else {
		runErr = c.daemon.AttachInteractive(
			context.Background(),
			c.id,
			c.stdin,
			c.stdout,
			c.stderr,
			"ctrl-p,ctrl-q",
		)
	}

	if c.waitForKey && runErr == nil {
		// Still in raw mode — single keypress is enough.
		_, _ = fmt.Fprintf(c.stdout, "\n\r\033[7m Press any key to return to dry... \033[0m")
		buf := make([]byte, 1)
		_, _ = c.stdin.Read(buf)
	}

	_ = term.Restore(fd, oldState)

	return runErr
}

// connectToDockerCmd connects to the Docker daemon asynchronously.
func connectToDockerCmd(cfg Config) tea.Cmd {
	return func() tea.Msg {
		env := docker.Env{
			DockerHost:      cfg.DockerHost,
			DockerCertPath:  cfg.DockerCertPath,
			DockerTLSVerify: cfg.DockerTLSVerify,
		}
		daemon, err := docker.ConnectToDaemon(env)
		if err != nil {
			return dockerErrorMsg{err: err}
		}
		return dockerConnectedMsg{daemon: daemon}
	}
}

// loadContainersCmd fetches the container list from Docker.
func loadContainersCmd(daemon docker.ContainerDaemon, showAll bool, sortMode docker.SortMode) tea.Cmd {
	return func() tea.Msg {
		var filters []docker.ContainerFilter
		if !showAll {
			filters = append(filters, docker.ContainerFilters.Running())
		}
		containers := daemon.Containers(filters, sortMode)
		return containersLoadedMsg{containers: containers}
	}
}

// listenDockerEvents blocks on the events channel and returns the next event.
func listenDockerEvents(ch <-chan events.Message) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return eventsClosedMsg{}
		}
		return dockerEventMsg{event: event}
	}
}

// loadingTickCmd returns a tick command for the loading animation.
func loadingTickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
		return loadingTickMsg{}
	})
}

// showHelpCmd opens the help text in the less viewer.
func showHelpCmd() tea.Cmd {
	return func() tea.Msg {
		return showLessMsg{
			content: ui.RenderMarkup(Help()),
			title:   "Help",
		}
	}
}

// inspectContainerCmd fetches container inspect JSON and shows it.
func inspectContainerCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		info, err := daemon.Inspect(id)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Inspect error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("JSON error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showLessMsg{
			content: string(data),
			title:   fmt.Sprintf("Inspect: %s", shortID(id)),
		}
	}
}

// formatEvent formats a Docker event as a display line.
func formatEvent(e events.Message) string {
	ts := time.Unix(0, e.TimeNano)
	actor := e.Actor.ID
	if actor == "" {
		actor = e.Actor.Attributes["name"]
	}
	return fmt.Sprintf("[%s] %s %s: %s", ts.Format("15:04:05"), e.Type, e.Action, actor)
}

// showDockerEventsCmd shows the Docker event log.
func showDockerEventsCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		eventLog := daemon.EventLog()
		if eventLog == nil {
			return showLessMsg{content: "No events recorded", title: "Docker Events"}
		}
		evts := eventLog.Events()
		var lines []string
		for _, e := range evts {
			lines = append(lines, formatEvent(e))
		}
		content := "No events recorded"
		if len(lines) > 0 {
			content = strings.Join(lines, "\n")
		}
		return showLessMsg{
			content: content,
			title:   "Docker Events",
		}
	}
}

// showDockerInfoCmd shows docker system info.
func showDockerInfoCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		info, err := daemon.Info()
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Info error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("JSON error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showLessMsg{
			content: string(data),
			title:   "Docker Info",
		}
	}
}

// loadImagesCmd fetches the image list from Docker.
func loadImagesCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		images, err := daemon.Images()
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Images error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return appui.ImagesLoadedMsg{Images: images}
	}
}

// loadNetworksCmd fetches the network list from Docker.
func loadNetworksCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		networks, err := daemon.Networks()
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Networks error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return appui.NetworksLoadedMsg{Networks: networks}
	}
}

// loadVolumesCmd fetches the volume list from Docker.
func loadVolumesCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		volumes, err := daemon.VolumeList(context.Background())
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Volumes error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return appui.VolumesLoadedMsg{Volumes: volumes}
	}
}

// loadDiskUsageCmd fetches Docker disk usage.
func loadDiskUsageCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		usage, err := daemon.DiskUsage()
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Disk usage error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return appui.DiskUsageLoadedMsg{Usage: usage}
	}
}

// inspectImageCmd fetches image inspect JSON and shows it.
func inspectImageCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		info, err := daemon.InspectImage(id)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Inspect error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("JSON error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showLessMsg{
			content: string(data),
			title:   fmt.Sprintf("Image Inspect: %s", shortID(id)),
		}
	}
}

// inspectNetworkCmd fetches network inspect JSON and shows it.
func inspectNetworkCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		info, err := daemon.NetworkInspect(id)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Inspect error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("JSON error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showLessMsg{
			content: string(data),
			title:   fmt.Sprintf("Network Inspect: %s", shortID(id)),
		}
	}
}

// showContainerLogsCmd opens a streaming log viewer for a container.
func showContainerLogsCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		stream, err := loadContainerLogStream(daemon, id)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Logs error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showStreamingLessMsg{
			content: stream.content,
			title:   stream.title,
			reader:  stream.reader,
		}
	}
}

// attachContainerCmd opens an interactive attach session for a running container.
func attachContainerCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	cmd := &interactiveExecCommand{daemon: daemon, id: id}
	return tea.Exec(cmd, func(err error) tea.Msg {
		if err != nil {
			return execEndedMsg{
				text:   fmt.Sprintf("Attach error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return execEndedMsg{
			text:   fmt.Sprintf("Attach ended: %s", shortID(id)),
			expiry: 3 * time.Second,
		}
	})
}

// isInteractiveShell returns true if the command looks like an interactive shell session.
func isInteractiveShell(command []string) bool {
	if len(command) == 0 {
		return false
	}
	shells := map[string]bool{
		"sh": true, "/bin/sh": true,
		"bash": true, "/bin/bash": true,
		"zsh": true, "/bin/zsh": true,
		"ash": true, "/bin/ash": true,
		"fish": true, "/usr/bin/fish": true,
		"ksh": true, "/bin/ksh": true,
		"csh": true, "/bin/csh": true,
		"tcsh": true, "/bin/tcsh": true,
	}
	return shells[command[0]]
}

// execContainerCmd opens an interactive exec session in a running container.
func execContainerCmd(daemon docker.ContainerDaemon, id string, command []string) tea.Cmd {
	waitForKey := !isInteractiveShell(command)
	cmd := &interactiveExecCommand{
		daemon:     daemon,
		id:         id,
		command:    command,
		waitForKey: waitForKey,
	}
	return tea.Exec(cmd, func(err error) tea.Msg {
		if err != nil {
			return execEndedMsg{
				text:   fmt.Sprintf("Exec error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return execEndedMsg{
			text:   fmt.Sprintf("Exec ended: %s", shortID(id)),
			expiry: 3 * time.Second,
		}
	})
}

// showContainerStatsCmd fetches a snapshot of container stats.
func showContainerStatsCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		c := daemon.ContainerByID(id)
		if c == nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Container %s not found", shortID(id)),
				expiry: 5 * time.Second,
			}
		}
		sc, err := daemon.StatsChannel(c)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Stats error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ch := sc.Start(ctx)
		stats := <-ch
		if stats == nil || stats.Error != nil {
			msg := "no stats available"
			if stats != nil && stats.Error != nil {
				msg = stats.Error.Error()
			}
			return statusMessageMsg{
				text:   fmt.Sprintf("Stats error: %s", msg),
				expiry: 5 * time.Second,
			}
		}
		data, _ := json.MarshalIndent(stats.Stats, "", "  ")
		content := string(data)
		if stats.ProcessList != nil {
			content += "\n\n--- Process List ---\n"
			for _, title := range stats.ProcessList.Titles {
				content += fmt.Sprintf("%-20s", title)
			}
			content += "\n"
			for _, proc := range stats.ProcessList.Processes {
				for _, field := range proc {
					content += fmt.Sprintf("%-20s", field)
				}
				content += "\n"
			}
		}
		return showLessMsg{
			content: content,
			title:   fmt.Sprintf("Stats: %s", shortID(id)),
		}
	}
}

// showImageHistoryCmd fetches image history and shows it.
func showImageHistoryCmd(daemon docker.ContainerDaemon, imageID string) tea.Cmd {
	return func() tea.Msg {
		history, err := daemon.History(imageID)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("History error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		data, _ := json.MarshalIndent(history, "", "  ")
		return showLessMsg{
			content: string(data),
			title:   fmt.Sprintf("Image History: %s", shortID(imageID)),
		}
	}
}

// inspectVolumeCmd fetches volume inspect JSON and shows it.
func inspectVolumeCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		info, err := daemon.VolumeInspect(context.Background(), id)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Inspect error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("JSON error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showLessMsg{
			content: string(data),
			title:   fmt.Sprintf("Volume: %s", id),
		}
	}
}

// --- Compose commands ---

// loadComposeProjectsCmd fetches compose projects with services derived from container labels.
func loadComposeProjectsCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		projects := daemon.ComposeProjectsWithServices()
		return appcompose.ProjectsLoadedMsg{Projects: projects}
	}
}

// loadComposeServicesCmd fetches compose resources (services, networks, volumes) for a project.
func loadComposeServicesCmd(daemon docker.ContainerDaemon, project string) tea.Cmd {
	return func() tea.Msg {
		services := daemon.ComposeServices(project)

		var composeNets []docker.ComposeNetwork
		if nets, err := daemon.Networks(); err == nil {
			for _, n := range nets {
				if n.Labels["com.docker.compose.project"] == project {
					composeNets = append(composeNets, docker.ComposeNetwork{
						Name:   n.Name,
						Driver: n.Driver,
						Scope:  n.Scope,
					})
				}
			}
		}

		var composeVols []docker.ComposeVolume
		if vols, err := daemon.VolumeList(context.Background()); err == nil {
			for _, v := range vols {
				if v.Labels["com.docker.compose.project"] == project {
					composeVols = append(composeVols, docker.ComposeVolume{
						Name:   v.Name,
						Driver: v.Driver,
					})
				}
			}
		}

		return appcompose.ServicesLoadedMsg{
			Services: services,
			Networks: composeNets,
			Volumes:  composeVols,
			Project:  project,
		}
	}
}

// inspectComposeServiceCmd finds the first container for a compose service and inspects it.
func inspectComposeServiceCmd(daemon docker.ContainerDaemon, project, service string) tea.Cmd {
	return func() tea.Msg {
		for _, c := range daemon.Containers(nil, docker.NoSort) {
			if c.Labels["com.docker.compose.project"] == project &&
				c.Labels["com.docker.compose.service"] == service {
				return inspectContainerCmd(daemon, c.ID)()
			}
		}
		return statusMessageMsg{
			text:   fmt.Sprintf("No containers found for service %s", service),
			expiry: 3 * time.Second,
		}
	}
}

// showComposeLogsCmd opens a merged streaming log viewer for compose
// containers matching the given project and optional service.
func showComposeLogsCmd(daemon docker.ContainerDaemon, project, service string) tea.Cmd {
	return func() tea.Msg {
		stream, err := loadComposeLogStream(daemon, project, service)
		if err != nil {
			return statusMessageMsg{
				text:   err.Error(),
				expiry: 3 * time.Second,
			}
		}
		return showStreamingLessMsg{
			content: stream.content,
			title:   stream.title,
			reader:  stream.reader,
		}
	}
}

func loadQuickPeekCmd(daemon docker.ContainerDaemon, ctx workspaceContext) tea.Cmd {
	return func() tea.Msg {
		msg := quickPeekLoadedMsg{
			title:       ctx.title,
			subtitle:    ctx.subtitle,
			detailTitle: "Preview",
			status:      "Preview ready",
			summary:     append([]string(nil), ctx.lines...),
		}

		switch ctx.kind {
		case workspaceContextContainer:
			stream, err := loadContainerLogStreamWithTail(daemon, ctx.containerID, quickPeekLogTail)
			if err != nil {
				msg.detailTitle = "Recent Logs"
				msg.status = "Unavailable · no recent container logs"
				msg.content = "No recent container logs available."
				return msg
			}
			content, err := collectQuickPeekLogContent(stream)
			if err != nil {
				msg.detailTitle = "Recent Logs"
				msg.status = "Read error · recent container logs"
				msg.content = fmt.Sprintf("Log read error: %s", err)
				return msg
			}
			msg.detailTitle = stream.title
			msg.status = fmt.Sprintf("Recent logs · last %d lines", quickPeekLogTail)
			msg.content = content
			if strings.TrimSpace(msg.content) == "" {
				msg.status = "Unavailable · no recent container logs"
				msg.content = "No recent container logs available."
			}
			return msg
		case workspaceContextComposeProject:
			stream, err := loadComposeLogStreamWithTail(daemon, ctx.project, "", quickPeekLogTail)
			if err != nil {
				msg.detailTitle = "Recent Logs"
				msg.status = "Unavailable · no recent project logs"
				msg.content = "No recent project logs available."
				return msg
			}
			content, err := collectQuickPeekLogContent(stream)
			if err != nil {
				msg.detailTitle = "Recent Logs"
				msg.status = "Read error · recent project logs"
				msg.content = fmt.Sprintf("Log read error: %s", err)
				return msg
			}
			msg.detailTitle = stream.title
			msg.status = fmt.Sprintf("Recent logs · last %d lines", quickPeekLogTail)
			msg.content = content
			if strings.TrimSpace(msg.content) == "" {
				msg.status = "Unavailable · no recent project logs"
				msg.content = "No recent project logs available."
			}
			return msg
		case workspaceContextComposeService:
			stream, err := loadComposeLogStreamWithTail(daemon, ctx.project, ctx.service, quickPeekLogTail)
			if err != nil {
				msg.detailTitle = "Recent Logs"
				msg.status = "Unavailable · no recent service logs"
				msg.content = "No recent service logs available."
				return msg
			}
			content, err := collectQuickPeekLogContent(stream)
			if err != nil {
				msg.detailTitle = "Recent Logs"
				msg.status = "Read error · recent service logs"
				msg.content = fmt.Sprintf("Log read error: %s", err)
				return msg
			}
			msg.detailTitle = stream.title
			msg.status = fmt.Sprintf("Recent logs · last %d lines", quickPeekLogTail)
			msg.content = content
			if strings.TrimSpace(msg.content) == "" {
				msg.status = "Unavailable · no recent service logs"
				msg.content = "No recent service logs available."
			}
			return msg
		}

		var activity workspaceActivityLoadedMsg
		switch {
		case ctx.imageID != "":
			activity = loadWorkspaceImageInspectCmd(daemon, ctx.imageID)().(workspaceActivityLoadedMsg)
		case ctx.monitorCID != "":
			activity = loadWorkspaceMonitorDetailsFromContext(ctx, 64, 12)().(workspaceActivityLoadedMsg)
		case ctx.networkID != "":
			activity = loadWorkspaceNetworkInspectCmd(daemon, ctx.networkID)().(workspaceActivityLoadedMsg)
		case ctx.nodeID != "":
			activity = loadWorkspaceNodeInspectCmd(daemon, ctx.nodeID)().(workspaceActivityLoadedMsg)
		case ctx.serviceID != "":
			activity = loadWorkspaceServiceInspectCmd(daemon, ctx.serviceID)().(workspaceActivityLoadedMsg)
		case ctx.stackName != "":
			activity = loadWorkspaceStackDetailsCmd(daemon, docker.Stack{
				Name:     ctx.stackName,
				Services: parseWorkspaceCount(ctx.lines, "services"),
				Networks: parseWorkspaceCount(ctx.lines, "networks"),
				Configs:  parseWorkspaceCount(ctx.lines, "configs"),
				Secrets:  parseWorkspaceCount(ctx.lines, "secrets"),
			})().(workspaceActivityLoadedMsg)
		case ctx.taskID != "":
			activity = loadWorkspaceTaskInspectCmd(daemon, ctx.taskID)().(workspaceActivityLoadedMsg)
		case ctx.volumeName != "":
			activity = loadWorkspaceVolumeInspectCmd(daemon, ctx.volumeName)().(workspaceActivityLoadedMsg)
		default:
			msg.status = "Summary only"
			msg.content = workspaceKeyValueContent(ctx.title, ctx.subtitle, ctx.lines)
			return msg
		}

		msg.detailTitle = activity.title
		msg.status = activity.status
		msg.content = activity.content
		return msg
	}
}

func loadWorkspaceActivityCmd(daemon docker.ContainerDaemon, ctx workspaceContext, activityWidth, activityHeight int) tea.Cmd {
	return func() tea.Msg {
		switch ctx.kind {
		case workspaceContextContainer:
			stream, err := loadContainerLogStream(daemon, ctx.containerID)
			if err != nil {
				return workspaceActivityLoadedMsg{
					title:   "Container Logs",
					status:  "Unavailable · pinned container has no live logs",
					content: "No live container logs available.",
				}
			}
			return workspaceActivityLoadedMsg{
				title:   stream.title,
				status:  "Live logs · follows pinned container",
				content: stream.content,
				reader:  stream.reader,
			}
		case workspaceContextComposeProject:
			stream, err := loadComposeLogStream(daemon, ctx.project, "")
			if err != nil {
				return workspaceActivityLoadedMsg{
					title:   "Project Logs",
					status:  "Unavailable · pinned project has no live logs",
					content: "No live project logs available.",
				}
			}
			return workspaceActivityLoadedMsg{
				title:   stream.title,
				status:  "Live logs · follows pinned project",
				content: stream.content,
				reader:  stream.reader,
			}
		case workspaceContextComposeService:
			stream, err := loadComposeLogStream(daemon, ctx.project, ctx.service)
			if err != nil {
				return workspaceActivityLoadedMsg{
					title:   "Service Logs",
					status:  "Unavailable · pinned service has no live logs",
					content: "No live service logs available.",
				}
			}
			return workspaceActivityLoadedMsg{
				title:   stream.title,
				status:  "Live logs · follows pinned service",
				content: stream.content,
				reader:  stream.reader,
			}
		case workspaceContextNone:
			return workspaceActivityLoadedMsg{
				title:   "Activity",
				status:  "Idle",
				content: "Pin a container or Compose project/service to stream logs here.",
			}
		default:
			if ctx.imageID != "" {
				return loadWorkspaceImageInspectCmd(daemon, ctx.imageID)()
			}
			if ctx.monitorCID != "" {
				return loadWorkspaceMonitorDetailsFromContext(ctx, activityWidth, activityHeight)()
			}
			if ctx.networkID != "" {
				return loadWorkspaceNetworkInspectCmd(daemon, ctx.networkID)()
			}
			if ctx.nodeID != "" {
				return loadWorkspaceNodeInspectCmd(daemon, ctx.nodeID)()
			}
			if ctx.serviceID != "" {
				return loadWorkspaceServiceInspectCmd(daemon, ctx.serviceID)()
			}
			if ctx.stackName != "" {
				return loadWorkspaceStackDetailsCmd(daemon, docker.Stack{
					Name:     ctx.stackName,
					Services: parseWorkspaceCount(ctx.lines, "services"),
					Networks: parseWorkspaceCount(ctx.lines, "networks"),
					Configs:  parseWorkspaceCount(ctx.lines, "configs"),
					Secrets:  parseWorkspaceCount(ctx.lines, "secrets"),
				})()
			}
			if ctx.taskID != "" {
				return loadWorkspaceTaskInspectCmd(daemon, ctx.taskID)()
			}
			if ctx.volumeName != "" {
				return loadWorkspaceVolumeInspectCmd(daemon, ctx.volumeName)()
			}
			return workspaceActivityLoadedMsg{
				title:   "Activity",
				status:  "Pinned context has no activity binding",
				content: "This pinned context only affects the context pane in Phase 1.",
			}
		}
	}
}

func loadWorkspaceImageInspectCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		info, err := daemon.InspectImage(id)
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Image Inspect",
				status:  "Inspect error · follows current image selection",
				content: fmt.Sprintf("Inspect error: %s", err),
			}
		}
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Image Inspect",
				status:  "JSON error · follows current image selection",
				content: fmt.Sprintf("JSON error: %s", err),
			}
		}
		return workspaceActivityLoadedMsg{
			title:   fmt.Sprintf("Image Inspect: %s", shortID(id)),
			status:  "Inspect · follows current image selection",
			content: string(data),
		}
	}
}

func loadWorkspaceNetworkInspectCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		info, err := daemon.NetworkInspect(id)
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Network Inspect",
				status:  "Inspect error · follows current network selection",
				content: fmt.Sprintf("Inspect error: %s", err),
			}
		}
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Network Inspect",
				status:  "JSON error · follows current network selection",
				content: fmt.Sprintf("JSON error: %s", err),
			}
		}
		return workspaceActivityLoadedMsg{
			title:   fmt.Sprintf("Network Inspect: %s", shortID(id)),
			status:  "Inspect · follows current network selection",
			content: string(data),
		}
	}
}

func loadWorkspaceVolumeInspectCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		info, err := daemon.VolumeInspect(context.Background(), id)
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Volume Inspect",
				status:  "Inspect error · follows current volume selection",
				content: fmt.Sprintf("Inspect error: %s", err),
			}
		}
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Volume Inspect",
				status:  "JSON error · follows current volume selection",
				content: fmt.Sprintf("JSON error: %s", err),
			}
		}
		return workspaceActivityLoadedMsg{
			title:   fmt.Sprintf("Volume Inspect: %s", id),
			status:  "Inspect · follows current volume selection",
			content: string(data),
		}
	}
}

func loadWorkspaceMonitorDetails(lookup monitorContainerLookup, stats *docker.Stats, series appui.MonitorSeries, chartWidth, chartHeight int) tea.Cmd {
	ctx := workspaceContextFromStats(stats, lookup, series)
	return loadWorkspaceMonitorDetailsFromContext(ctx, chartWidth, chartHeight)
}

func loadWorkspaceMonitorDetailsFromContext(ctx workspaceContext, chartWidth, chartHeight int) tea.Cmd {
	return func() tea.Msg {
		cpuHistory := ctx.monitorHistory(ctx.monitorCPUHistory, ctx.monitorCPU)
		content := workspaceMonitorDetailContent(ctx, chartWidth, chartHeight)
		return workspaceActivityLoadedMsg{
			title:   fmt.Sprintf("Monitor Details: %s", ctx.title),
			status:  fmt.Sprintf("Live stats · %s/%s window", formatMonitorDuration(monitorCollectedDuration(cpuHistory, monitorChartWindow)), formatMonitorDuration(monitorChartWindow)),
			content: content,
		}
	}
}

func workspaceMonitorDetailContent(ctx workspaceContext, chartWidth, chartHeight int) string {
	cpuHistory := ctx.monitorHistory(ctx.monitorCPUHistory, ctx.monitorCPU)
	memHistory := ctx.monitorHistory(ctx.monitorMemHistory, ctx.monitorPct)
	halfWidth := chartWidth / 2
	cpu := monitorHistorySection(
		"CPU",
		cpuHistory,
		fmt.Sprintf("now %5.1f%%", ctx.monitorCPU),
		appui.DryTheme.Info,
		halfWidth,
		chartHeight,
		runes.ArcLineStyle,
	)
	mem := monitorHistorySection(
		"Memory",
		memHistory,
		fmt.Sprintf("%s / %s  (%5.1f%%)", units.BytesSize(ctx.monitorMem), units.BytesSize(ctx.monitorMax), ctx.monitorPct),
		appui.DryTheme.Secondary,
		halfWidth,
		chartHeight,
		runes.ThinLineStyle,
	)
	indent := lipgloss.NewStyle().PaddingLeft(2)
	cpuPane := lipgloss.PlaceHorizontal(halfWidth, lipgloss.Left, indent.Render(cpu))
	memPane := lipgloss.PlaceHorizontal(halfWidth, lipgloss.Left, indent.Render(mem))
	return monitorLegendLine() + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top, cpuPane, memPane)
}

func (ctx workspaceContext) monitorHistory(history []appui.MonitorPoint, current float64) []appui.MonitorPoint {
	if len(history) > 0 {
		return history
	}
	return []appui.MonitorPoint{{At: time.Now(), Value: current}}
}

func monitorHistorySection(title string, samples []appui.MonitorPoint, detail string, accent color.Color, chartWidth, bodyHeight int, lineStyle runes.LineStyle) string {
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(appui.DryTheme.Fg)
	detailStyle := lipgloss.NewStyle().Foreground(appui.DryTheme.FgMuted)
	graph := monitorHistoryChart(samples, monitorChartWidth(chartWidth), monitorChartHeight(bodyHeight), accent, lineStyle)
	return strings.Join([]string{
		labelStyle.Render(title),
		detailStyle.Render(detail),
		graph,
	}, "\n")
}

func monitorHistoryChart(samples []appui.MonitorPoint, width, height int, accent color.Color, lineStyle runes.LineStyle) string {
	if len(samples) == 0 {
		samples = []appui.MonitorPoint{{At: time.Now(), Value: 0}}
	}
	samples = trimMonitorHistory(samples, monitorChartWindow)
	minTime := samples[0].At
	maxTime := samples[len(samples)-1].At
	if !maxTime.After(minTime) {
		maxTime = minTime.Add(time.Second)
	}
	minY, maxY := monitorChartYRange(samples)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(colorToHex(accent)))
	axisStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorToHex(appui.DryTheme.FgSubtle)))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorToHex(appui.DryTheme.FgMuted)))
	chart := timeserieslinechart.New(
		width,
		height,
		timeserieslinechart.WithTimeRange(minTime, maxTime),
		timeserieslinechart.WithYRange(minY, maxY),
		timeserieslinechart.WithXLabelFormatter(timeserieslinechart.HourTimeLabelFormatter()),
		timeserieslinechart.WithXYSteps(0, 2),
		timeserieslinechart.WithLineStyle(lineStyle),
		timeserieslinechart.WithStyle(style),
		timeserieslinechart.WithAxesStyles(axisStyle, labelStyle),
	)
	for _, sample := range samples {
		chart.Push(timeserieslinechart.TimePoint{
			Time:  sample.At,
			Value: sample.Value,
		})
	}
	chart.DrawBraille()
	return strings.TrimRight(chart.View(), "\n")
}

func monitorLegendLine() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(appui.DryTheme.FgSubtle)
	cpuStyle := lipgloss.NewStyle().
		Foreground(appui.DryTheme.Info)
	memStyle := lipgloss.NewStyle().
		Foreground(appui.DryTheme.Secondary)
	return strings.Join([]string{
		labelStyle.Render("Legend:"),
		cpuStyle.Render("CPU arc"),
		memStyle.Render("Memory line"),
		labelStyle.Render("Y auto-scaled"),
	}, labelStyle.Render("  ·  "))
}

func trimMonitorHistory(samples []appui.MonitorPoint, window time.Duration) []appui.MonitorPoint {
	if len(samples) == 0 {
		return nil
	}
	cutoff := samples[len(samples)-1].At.Add(-window)
	start := 0
	for start < len(samples)-1 && samples[start].At.Before(cutoff) {
		start++
	}
	return append([]appui.MonitorPoint(nil), samples[start:]...)
}

func monitorCollectedDuration(samples []appui.MonitorPoint, limit time.Duration) time.Duration {
	if len(samples) < 2 {
		return 0
	}
	d := samples[len(samples)-1].At.Sub(samples[0].At)
	if d < 0 {
		return 0
	}
	if d > limit {
		return limit
	}
	return d
}

func formatMonitorDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	if d <= 0 {
		return "0s"
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	if d >= time.Minute {
		return fmt.Sprintf("%dm%02ds", int(d/time.Minute), int((d%time.Minute)/time.Second))
	}
	return fmt.Sprintf("%ds", int(d/time.Second))
}

func monitorChartYRange(samples []appui.MonitorPoint) (float64, float64) {
	if len(samples) == 0 {
		return 0, 100
	}
	minY := samples[0].Value
	maxY := samples[0].Value
	for _, sample := range samples[1:] {
		if sample.Value < minY {
			minY = sample.Value
		}
		if sample.Value > maxY {
			maxY = sample.Value
		}
	}

	span := maxY - minY
	if span < 5 {
		center := (maxY + minY) / 2
		minY = center - 2.5
		maxY = center + 2.5
	} else {
		padding := math.Max(1, span*0.15)
		minY -= padding
		maxY += padding
	}

	if minY < 0 {
		minY = 0
	}
	if maxY > 100 {
		maxY = 100
	}
	if maxY-minY < 1 {
		maxY = math.Min(100, minY+1)
	}
	return minY, maxY
}

func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func monitorChartWidth(halfWidth int) int {
	if halfWidth <= 0 {
		return minChartWidth + chartPadding
	}
	return max(minChartWidth, min(halfWidth-chartPadding, maxChartWidth))
}

func monitorChartHeight(bodyHeight int) int {
	available := bodyHeight - chartOverhead
	height := max(minChartHeight, available*chartHeightPct/100)
	if height > available {
		height = available
	}
	return max(minChartHeight, height)
}

func loadWorkspaceNodeInspectCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		node, err := daemon.Node(id)
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Node Inspect",
				status:  "Inspect error · follows current node selection",
				content: fmt.Sprintf("Inspect error: %s", err),
			}
		}
		data, err := json.MarshalIndent(node, "", "  ")
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Node Inspect",
				status:  "JSON error · follows current node selection",
				content: fmt.Sprintf("JSON error: %s", err),
			}
		}
		return workspaceActivityLoadedMsg{
			title:   fmt.Sprintf("Node Inspect: %s", shortID(id)),
			status:  "Inspect · follows current node selection",
			content: string(data),
		}
	}
}

func loadWorkspaceServiceInspectCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		svc, err := daemon.Service(id)
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Service Inspect",
				status:  "Inspect error · follows current service selection",
				content: fmt.Sprintf("Inspect error: %s", err),
			}
		}
		data, err := json.MarshalIndent(svc, "", "  ")
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Service Inspect",
				status:  "JSON error · follows current service selection",
				content: fmt.Sprintf("JSON error: %s", err),
			}
		}
		return workspaceActivityLoadedMsg{
			title:   fmt.Sprintf("Service Inspect: %s", shortID(id)),
			status:  "Inspect · follows current service selection",
			content: string(data),
		}
	}
}

func loadWorkspaceTaskInspectCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		task, err := daemon.Task(id)
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Task Inspect",
				status:  "Inspect error · follows current task selection",
				content: fmt.Sprintf("Inspect error: %s", err),
			}
		}
		data, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			return workspaceActivityLoadedMsg{
				title:   "Task Inspect",
				status:  "JSON error · follows current task selection",
				content: fmt.Sprintf("JSON error: %s", err),
			}
		}
		return workspaceActivityLoadedMsg{
			title:   fmt.Sprintf("Task Inspect: %s", shortID(id)),
			status:  "Inspect · follows current task selection",
			content: string(data),
		}
	}
}

func loadWorkspaceStackDetailsCmd(daemon docker.ContainerDaemon, stack docker.Stack) tea.Cmd {
	return func() tea.Msg {
		var lines []string
		lines = append(lines, fmt.Sprintf("name: %s", stack.Name))
		lines = append(lines, fmt.Sprintf("services: %d", stack.Services))

		networks, netErr := daemon.StackNetworks(stack.Name)
		if netErr != nil {
			lines = append(lines, fmt.Sprintf("network inspect error: %s", netErr))
		} else {
			lines = append(lines, fmt.Sprintf("networks: %d", len(networks)))
			for _, n := range networks {
				if n.Name != "" {
					lines = append(lines, fmt.Sprintf("network: %s", n.Name))
				}
			}
		}

		configs, cfgErr := daemon.StackConfigs(stack.Name)
		if cfgErr != nil {
			lines = append(lines, fmt.Sprintf("config inspect error: %s", cfgErr))
		} else {
			lines = append(lines, fmt.Sprintf("configs: %d", len(configs)))
			for _, c := range configs {
				if c.Spec.Name != "" {
					lines = append(lines, fmt.Sprintf("config: %s", c.Spec.Name))
				}
			}
		}

		secrets, secErr := daemon.StackSecrets(stack.Name)
		if secErr != nil {
			lines = append(lines, fmt.Sprintf("secret inspect error: %s", secErr))
		} else {
			lines = append(lines, fmt.Sprintf("secrets: %d", len(secrets)))
			for _, s := range secrets {
				if s.Spec.Name != "" {
					lines = append(lines, fmt.Sprintf("secret: %s", s.Spec.Name))
				}
			}
		}

		tasks, taskErr := daemon.StackTasks(stack.Name)
		if taskErr != nil {
			lines = append(lines, fmt.Sprintf("task inspect error: %s", taskErr))
		} else {
			lines = append(lines, fmt.Sprintf("tasks: %d", len(tasks)))
			for _, t := range tasks {
				lines = append(lines, fmt.Sprintf("task: %s (%s)", shortID(t.ID), t.Status.State))
			}
		}

		return workspaceActivityLoadedMsg{
			title:   fmt.Sprintf("Stack Details: %s", stack.Name),
			status:  "Details · follows current stack selection",
			content: workspaceKeyValueContent(stack.Name, "Stack", lines),
		}
	}
}

func workspaceKeyValueContent(title, subtitle string, lines []string) string {
	parts := make([]string, 0, len(lines)+2)
	if title != "" {
		parts = append(parts, title)
	}
	if subtitle != "" {
		parts = append(parts, subtitle)
	}
	if len(parts) > 0 && len(lines) > 0 {
		parts = append(parts, "")
	}
	parts = append(parts, lines...)
	return strings.Join(parts, "\n")
}

func parseWorkspaceCount(lines []string, key string) int {
	for _, line := range lines {
		prefix := key + ": "
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		var value int
		if _, err := fmt.Sscanf(strings.TrimPrefix(line, prefix), "%d", &value); err == nil {
			return value
		}
	}
	return 0
}

// mergeLogReaders multiplexes multiple named log readers into a single
// reader, prefixing each line with its name (e.g. "api | <line>").
func mergeLogReaders(named map[string]io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, reader := range named {
		wg.Add(1)
		go func(prefix string, r io.ReadCloser) {
			defer wg.Done()
			defer func() { _ = r.Close() }()
			scanner := bufio.NewScanner(r)
			scanner.Buffer(make([]byte, 64*1024), 1024*1024)
			for scanner.Scan() {
				line := fmt.Sprintf("%s | %s\n", prefix, scanner.Text())
				mu.Lock()
				_, err := pw.Write([]byte(line))
				mu.Unlock()
				if err != nil {
					return
				}
			}
		}(name, reader)
	}

	go func() {
		wg.Wait()
		_ = pw.Close()
	}()

	return &demuxedReadCloser{
		Reader: pr,
		close:  pr.Close,
	}
}

// --- Swarm commands ---

// loadNodesCmd fetches the swarm node list.
func loadNodesCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		nodes, err := daemon.Nodes()
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Nodes error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return appswarm.NodesLoadedMsg{Nodes: nodes}
	}
}

// loadServicesCmd fetches the swarm service list.
func loadServicesCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		services, err := daemon.Services()
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Services error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return appswarm.ServicesLoadedMsg{Services: services}
	}
}

// loadStacksCmd fetches the swarm stack list.
func loadStacksCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		stacks, err := daemon.Stacks()
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Stacks error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return appswarm.StacksLoadedMsg{Stacks: stacks}
	}
}

// loadNodeTasksCmd loads tasks for a specific node.
func loadNodeTasksCmd(daemon docker.ContainerDaemon, nodeID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := daemon.NodeTasks(nodeID)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Node tasks error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return appswarm.TasksLoadedMsg{
			Tasks: tasks,
			Title: fmt.Sprintf("Tasks for Node %s", shortID(nodeID)),
		}
	}
}

// loadServiceTasksCmd loads tasks for a specific service.
func loadServiceTasksCmd(daemon docker.ContainerDaemon, serviceID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := daemon.ServiceTasks(serviceID)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Service tasks error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return appswarm.TasksLoadedMsg{
			Tasks: tasks,
			Title: fmt.Sprintf("Tasks for Service %s", shortID(serviceID)),
		}
	}
}

// loadStackTasksCmd loads tasks for a specific stack.
func loadStackTasksCmd(daemon docker.ContainerDaemon, stackName string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := daemon.StackTasks(stackName)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Stack tasks error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return appswarm.TasksLoadedMsg{
			Tasks: tasks,
			Title: fmt.Sprintf("Tasks for Stack %s", stackName),
		}
	}
}

// inspectNodeCmd shows node inspect JSON.
func inspectNodeCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		node, err := daemon.Node(id)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Inspect error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		data, err := json.MarshalIndent(node, "", "  ")
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("JSON error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showLessMsg{
			content: string(data),
			title:   fmt.Sprintf("Node: %s", shortID(id)),
		}
	}
}

// inspectServiceCmd shows service inspect JSON.
func inspectServiceCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		svc, err := daemon.Service(id)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Inspect error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		data, err := json.MarshalIndent(svc, "", "  ")
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("JSON error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showLessMsg{
			content: string(data),
			title:   fmt.Sprintf("Service: %s", shortID(id)),
		}
	}
}

// inspectTaskCmd shows task inspect JSON.
func inspectTaskCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		task, err := daemon.Task(id)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Inspect error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		data, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("JSON error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return showLessMsg{
			content: string(data),
			title:   fmt.Sprintf("Task: %s", shortID(id)),
		}
	}
}

// showServiceLogsCmd opens a streaming log viewer for a service.
func showServiceLogsCmd(daemon docker.ContainerDaemon, id string) tea.Cmd {
	return func() tea.Msg {
		reader, err := daemon.ServiceLogs(id, "", false)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Logs error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		demuxed := demuxDockerStream(reader)
		buf := make([]byte, 64*1024)
		n, _ := demuxed.Read(buf)
		return showStreamingLessMsg{
			content: string(buf[:n]),
			title:   fmt.Sprintf("Service Logs: %s", shortID(id)),
			reader:  demuxed,
		}
	}
}
