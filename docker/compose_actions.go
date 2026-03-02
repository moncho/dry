package docker

import (
	"fmt"
	"sort"
)

// ComposeServiceActionReport summarises the result of a lifecycle action on a
// Compose service's containers.
type ComposeServiceActionReport struct {
	Project   string
	Service   string
	Action    string // "start", "stop", "restart", "remove"
	Targeted  int
	Attempted int
	Succeeded int
	Failed    int
	Skipped   int
	Errors    []string
}

// Summary formats the report as a one-line status message.
func (r ComposeServiceActionReport) Summary() string {
	action := r.Action
	if len(action) > 0 {
		action = string(action[0]-32) + action[1:]
	}
	target := r.Project + "/" + r.Service
	if r.Service == "" {
		target = r.Project
	}
	base := fmt.Sprintf("%s %s: %d targeted, %d succeeded",
		action, target, r.Targeted, r.Succeeded)
	if r.Skipped > 0 {
		base += fmt.Sprintf(", %d skipped", r.Skipped)
	}
	if r.Failed > 0 {
		base += fmt.Sprintf(", %d failed", r.Failed)
	}
	return base
}

// composeServiceContainers returns all containers belonging to the given
// Compose project+service, sorted by name then ID for deterministic ordering.
func composeServiceContainers(daemon *DockerDaemon, project, service string) []*Container {
	all := daemon.Containers(nil, NoSort)
	var matched []*Container
	for _, c := range all {
		if c.Labels["com.docker.compose.project"] != project {
			continue
		}
		if c.Labels["com.docker.compose.service"] != service {
			continue
		}
		if c.Labels["com.docker.compose.oneoff"] == "True" {
			continue
		}
		matched = append(matched, c)
	}
	sort.Slice(matched, func(i, j int) bool {
		ni := ""
		nj := ""
		if len(matched[i].Names) > 0 {
			ni = matched[i].Names[0]
		}
		if len(matched[j].Names) > 0 {
			nj = matched[j].Names[0]
		}
		if ni != nj {
			return ni < nj
		}
		return matched[i].ID < matched[j].ID
	})
	return matched
}

// ComposeServiceStart starts non-running containers of the given Compose service.
func (daemon *DockerDaemon) ComposeServiceStart(project, service string) (ComposeServiceActionReport, error) {
	targets := composeServiceContainers(daemon, project, service)
	report := ComposeServiceActionReport{
		Project:  project,
		Service:  service,
		Action:   "start",
		Targeted: len(targets),
	}
	for _, c := range targets {
		if IsContainerRunning(c) {
			report.Skipped++
			continue
		}
		report.Attempted++
		if err := daemon.StartContainer(c.ID); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", c.ID[:12], err))
		} else {
			report.Succeeded++
		}
	}
	if err := daemon.refreshAndWait(); err != nil {
		return report, err
	}
	return report, nil
}

// ComposeServiceStop stops running containers of the given Compose service.
func (daemon *DockerDaemon) ComposeServiceStop(project, service string) (ComposeServiceActionReport, error) {
	targets := composeServiceContainers(daemon, project, service)
	report := ComposeServiceActionReport{
		Project:  project,
		Service:  service,
		Action:   "stop",
		Targeted: len(targets),
	}
	for _, c := range targets {
		if !IsContainerRunning(c) {
			report.Skipped++
			continue
		}
		report.Attempted++
		if err := daemon.StopContainer(c.ID); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", c.ID[:12], err))
		} else {
			report.Succeeded++
		}
	}
	if err := daemon.refreshAndWait(); err != nil {
		return report, err
	}
	return report, nil
}

// ComposeServiceRestart restarts all containers of the given Compose service.
func (daemon *DockerDaemon) ComposeServiceRestart(project, service string) (ComposeServiceActionReport, error) {
	targets := composeServiceContainers(daemon, project, service)
	report := ComposeServiceActionReport{
		Project:  project,
		Service:  service,
		Action:   "restart",
		Targeted: len(targets),
	}
	for _, c := range targets {
		report.Attempted++
		if err := daemon.RestartContainer(c.ID); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", c.ID[:12], err))
		} else {
			report.Succeeded++
		}
	}
	if err := daemon.refreshAndWait(); err != nil {
		return report, err
	}
	return report, nil
}

// ComposeServiceRemove force-removes all containers of the given Compose service.
func (daemon *DockerDaemon) ComposeServiceRemove(project, service string) (ComposeServiceActionReport, error) {
	targets := composeServiceContainers(daemon, project, service)
	report := ComposeServiceActionReport{
		Project:  project,
		Service:  service,
		Action:   "remove",
		Targeted: len(targets),
	}
	for _, c := range targets {
		report.Attempted++
		if err := daemon.Rm(c.ID); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", c.ID[:12], err))
		} else {
			report.Succeeded++
		}
	}
	if err := daemon.refreshAndWait(); err != nil {
		return report, err
	}
	return report, nil
}

// composeProjectContainers returns all containers belonging to the given
// Compose project, sorted by name then ID for deterministic ordering.
func composeProjectContainers(daemon *DockerDaemon, project string) []*Container {
	all := daemon.Containers(nil, NoSort)
	var matched []*Container
	for _, c := range all {
		if c.Labels["com.docker.compose.project"] != project {
			continue
		}
		if c.Labels["com.docker.compose.oneoff"] == "True" {
			continue
		}
		matched = append(matched, c)
	}
	sort.Slice(matched, func(i, j int) bool {
		ni := ""
		nj := ""
		if len(matched[i].Names) > 0 {
			ni = matched[i].Names[0]
		}
		if len(matched[j].Names) > 0 {
			nj = matched[j].Names[0]
		}
		if ni != nj {
			return ni < nj
		}
		return matched[i].ID < matched[j].ID
	})
	return matched
}

// ComposeProjectStop stops running containers of the given Compose project.
func (daemon *DockerDaemon) ComposeProjectStop(project string) (ComposeServiceActionReport, error) {
	targets := composeProjectContainers(daemon, project)
	report := ComposeServiceActionReport{
		Project:  project,
		Action:   "stop",
		Targeted: len(targets),
	}
	for _, c := range targets {
		if !IsContainerRunning(c) {
			report.Skipped++
			continue
		}
		report.Attempted++
		if err := daemon.StopContainer(c.ID); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", c.ID[:12], err))
		} else {
			report.Succeeded++
		}
	}
	if err := daemon.refreshAndWait(); err != nil {
		return report, err
	}
	return report, nil
}

// ComposeProjectRestart restarts all containers of the given Compose project.
func (daemon *DockerDaemon) ComposeProjectRestart(project string) (ComposeServiceActionReport, error) {
	targets := composeProjectContainers(daemon, project)
	report := ComposeServiceActionReport{
		Project:  project,
		Action:   "restart",
		Targeted: len(targets),
	}
	for _, c := range targets {
		report.Attempted++
		if err := daemon.RestartContainer(c.ID); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", c.ID[:12], err))
		} else {
			report.Succeeded++
		}
	}
	if err := daemon.refreshAndWait(); err != nil {
		return report, err
	}
	return report, nil
}

// ComposeProjectRemove force-removes all containers of the given Compose project.
func (daemon *DockerDaemon) ComposeProjectRemove(project string) (ComposeServiceActionReport, error) {
	targets := composeProjectContainers(daemon, project)
	report := ComposeServiceActionReport{
		Project:  project,
		Action:   "remove",
		Targeted: len(targets),
	}
	for _, c := range targets {
		report.Attempted++
		if err := daemon.Rm(c.ID); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", c.ID[:12], err))
		} else {
			report.Succeeded++
		}
	}
	if err := daemon.refreshAndWait(); err != nil {
		return report, err
	}
	return report, nil
}
