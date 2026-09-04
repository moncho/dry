package app

// Container, image, network, volume, swarm, and compose operations executed
// from prompts, menus, and the command palette. Moved verbatim from model.go.

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/docker/go-units"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

func (m model) executeMenuCommand(containerID string, cmd docker.Command) (model, tea.Cmd) {
	switch cmd {
	case docker.INSPECT:
		return m, inspectContainerCmd(m.daemon, containerID)
	case docker.LOGS:
		return m, showContainerLogsCmd(m.daemon, containerID)
	case docker.ATTACH:
		return m, attachContainerCmd(m.daemon, containerID)
	case docker.EXEC:
		var cmd tea.Cmd
		m.inputPrompt, cmd = appui.NewInputPromptModelWithLimit(
			fmt.Sprintf("Exec in %s:", shortID(containerID)),
			"/bin/sh", "exec", containerID, 120,
		)
		m.inputPrompt.SetSize(m.width, m.height)
		m.overlay = overlayInputPrompt
		return m, cmd
	case docker.KILL:
		return m.showPrompt(
			fmt.Sprintf("Kill container %s?", shortID(containerID)),
			"kill", containerID,
		), nil
	case docker.STOP:
		return m.showPrompt(
			fmt.Sprintf("Stop container %s?", shortID(containerID)),
			"stop", containerID,
		), nil
	case docker.RESTART:
		return m.showPrompt(
			fmt.Sprintf("Restart container %s?", shortID(containerID)),
			"restart", containerID,
		), nil
	case docker.RM:
		return m.showPrompt(
			fmt.Sprintf("Remove container %s?", shortID(containerID)),
			"rm", containerID,
		), nil
	case docker.STATS:
		return m, showContainerStatsCmd(m.daemon, containerID)
	case docker.HISTORY:
		if c := m.daemon.ContainerByID(containerID); c != nil {
			return m, showImageHistoryCmd(m.daemon, c.ImageID)
		}
		return m, nil
	}
	return m, nil
}

func (m model) executeContainerOp(tag, id string) tea.Cmd {
	daemon := m.daemon
	return func() tea.Msg {
		var err error
		var successMsg string
		switch tag {
		case "kill":
			err = daemon.Kill(id)
			successMsg = fmt.Sprintf("Container %s killed", shortID(id))
		case "stop":
			err = daemon.StopContainer(id)
			successMsg = fmt.Sprintf("Container %s stopped", shortID(id))
		case "restart":
			err = daemon.RestartContainer(id)
			successMsg = fmt.Sprintf("Container %s restarted", shortID(id))
		case "rm":
			err = daemon.Rm(id)
			successMsg = fmt.Sprintf("Container %s removed", shortID(id))
		case "rm-all-stopped":
			var count int
			count, err = daemon.RemoveAllStoppedContainers()
			successMsg = fmt.Sprintf("Removed %d stopped containers", count)
		case "rmi":
			_, err = daemon.Rmi(id, false)
			successMsg = "Image removed"
		case "rmi-force":
			_, err = daemon.Rmi(id, true)
			successMsg = "Image force removed"
		case "rmi-dangling":
			var count int
			count, err = daemon.RemoveDanglingImages()
			successMsg = fmt.Sprintf("Removed %d dangling images", count)
		case "rmi-unused":
			var count int
			count, err = daemon.RemoveUnusedImages()
			successMsg = fmt.Sprintf("Removed %d unused images", count)
		case "net-rm":
			err = daemon.RemoveNetwork(id)
			successMsg = "Network removed"
		case "vol-rm":
			err = daemon.VolumeRemove(context.Background(), id, false)
			successMsg = fmt.Sprintf("Volume %s removed", id)
		case "vol-rm-force":
			err = daemon.VolumeRemove(context.Background(), id, true)
			successMsg = fmt.Sprintf("Volume %s force removed", id)
		case "vol-rm-all":
			var count int
			count, err = daemon.VolumeRemoveAll(context.Background())
			successMsg = fmt.Sprintf("Removed %d volumes", count)
		case "vol-prune":
			var count int
			count, err = daemon.VolumePrune(context.Background())
			successMsg = fmt.Sprintf("Pruned %d unused volumes", count)
		case "service-rm":
			err = daemon.ServiceRemove(id)
			successMsg = fmt.Sprintf("Service %s removed", shortID(id))
		case "service-update":
			err = daemon.ServiceUpdate(id)
			successMsg = fmt.Sprintf("Service %s update forced", shortID(id))
		case "stack-rm":
			err = daemon.StackRemove(id)
			successMsg = fmt.Sprintf("Stack %s removed", id)
		case "compose-start":
			project, service := splitComposeServiceID(id)
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeServiceStart(project, service)
			successMsg = report.Summary()
		case "compose-stop":
			project, service := splitComposeServiceID(id)
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeServiceStop(project, service)
			successMsg = report.Summary()
		case "compose-restart":
			project, service := splitComposeServiceID(id)
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeServiceRestart(project, service)
			successMsg = report.Summary()
		case "compose-rm":
			project, service := splitComposeServiceID(id)
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeServiceRemove(project, service)
			successMsg = report.Summary()
		case "compose-project-stop":
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeProjectStop(id)
			successMsg = report.Summary()
		case "compose-project-restart":
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeProjectRestart(id)
			successMsg = report.Summary()
		case "compose-project-rm":
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeProjectRemove(id)
			successMsg = report.Summary()
		case "compose-project-up":
			// The two compose actions that stream into the viewer rather
			// than reporting a summary, so they return the command's
			// message directly.
			return composeUpCmd(m.composeCLI, m.composeProjectFor(id))()
		case "compose-project-down":
			return composeDownCmd(m.composeCLI, m.composeProjectFor(id))()
		case "prune":
			report, pruneErr := daemon.Prune()
			if pruneErr != nil {
				err = pruneErr
			} else {
				successMsg = fmt.Sprintf("Pruned: %d containers, %d images, %d networks, %d volumes, reclaimed %s",
					len(report.ContainerReport.ContainersDeleted),
					len(report.ImagesReport.ImagesDeleted),
					len(report.NetworksReport.NetworksDeleted),
					len(report.VolumesReport.VolumesDeleted),
					units.BytesSize(float64(report.TotalSpaceReclaimed())))
			}
		default:
			return nil
		}
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return operationSuccessMsg{message: successMsg}
	}
}

func (m model) executeInputOp(tag, id, value string) tea.Cmd {
	daemon := m.daemon
	switch tag {
	case "exec":
		value = strings.TrimSpace(value)
		if value == "" {
			value = "/bin/sh"
		}
		command := strings.Fields(value)
		return execContainerCmd(daemon, id, command)
	case "service-scale":
		var replicas uint64
		if _, err := fmt.Sscanf(value, "%d", &replicas); err != nil {
			return func() tea.Msg {
				return statusMessageMsg{
					text:   fmt.Sprintf("Invalid replica count: %s", value),
					expiry: 5 * time.Second,
				}
			}
		}
		return func() tea.Msg {
			err := daemon.ServiceScale(id, replicas)
			if err != nil {
				return statusMessageMsg{
					text:   fmt.Sprintf("Scale error: %s", err),
					expiry: 5 * time.Second,
				}
			}
			return operationSuccessMsg{
				message: fmt.Sprintf("Service %s scaled to %d replicas", shortID(id), replicas),
			}
		}
	}
	return nil
}

func (m model) cycleNodeAvailability(nodeID string) tea.Cmd {
	daemon := m.daemon
	return func() tea.Msg {
		node, err := daemon.Node(nodeID)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Node error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		// Cycle: active → pause → drain → active
		var next swarm.NodeAvailability
		switch node.Spec.Availability {
		case swarm.NodeAvailabilityActive:
			next = swarm.NodeAvailabilityPause
		case swarm.NodeAvailabilityPause:
			next = swarm.NodeAvailabilityDrain
		default:
			next = swarm.NodeAvailabilityActive
		}
		err = daemon.NodeChangeAvailability(nodeID, next)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Availability error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return operationSuccessMsg{
			message: fmt.Sprintf("Node %s availability set to %s", shortID(nodeID), next),
		}
	}
}

// composeServiceID pairs a project with one of its services, for a
// confirmation prompt that has to survive until the user answers it. Both
// names come from container labels anything can set, so the halves are
// joined by a NUL rather than a slash: joined by a slash, project "a"
// service "b/c" and project "a/b" service "c" are the same id, and the
// confirmation then acts on a service the user did not name.
func composeServiceID(project, service string) string { return project + "\x00" + service }

// splitComposeServiceID undoes composeServiceID.
func splitComposeServiceID(id string) (project, service string) {
	project, service, _ = strings.Cut(id, "\x00")
	return project, service
}
