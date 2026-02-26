package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/moncho/dry/appui"
	appswarm "github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

// demuxedReadCloser wraps a pipe reader and closes the underlying Docker stream.
type demuxedReadCloser struct {
	io.Reader
	close func() error
}

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
			pw.Close()
		}
		raw.Close()
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
			reader.Close()
			return streamClosedMsg{}
		}
		return streamClosedMsg{}
	}
}

// shortID safely truncates an ID to at most 12 characters.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
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

// showDockerEventsCmd shows the Docker event log.
func showDockerEventsCmd(daemon docker.ContainerDaemon) tea.Cmd {
	return func() tea.Msg {
		eventLog := daemon.EventLog()
		events := eventLog.Events()
		var lines []string
		for _, e := range events {
			ts := time.Unix(e.Time, e.TimeNano)
			lines = append(lines, fmt.Sprintf("[%s] %s %s: %s",
				ts.Format("15:04:05"), e.Type, e.Action, e.Actor.ID))
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
		reader, err := daemon.Logs(id, "", false)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Logs error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		demuxed := demuxDockerStream(reader)
		// Read initial chunk to show immediately.
		buf := make([]byte, 64*1024)
		n, _ := demuxed.Read(buf)
		return showStreamingLessMsg{
			content: string(buf[:n]),
			title:   fmt.Sprintf("Logs: %s", shortID(id)),
			reader:  demuxed,
		}
	}
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
