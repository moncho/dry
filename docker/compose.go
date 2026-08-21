package docker

import (
	"fmt"
	"sort"
	"strings"

	"github.com/moby/moby/api/types/container"
)

// ProjectStatus describes whether a Compose project is up.
type ProjectStatus string

const (
	// ProjectRunning means at least one container is running.
	ProjectRunning ProjectStatus = "running"
	// ProjectStopped means containers exist but none are running.
	ProjectStopped ProjectStatus = "stopped"
	// ProjectNotCreated means the project is known only from its file.
	ProjectNotCreated ProjectStatus = "not created"
)

// ComposeProject represents a Docker Compose project aggregated from container
// labels, or discovered from a compose file when it has no containers yet.
type ComposeProject struct {
	Name       string
	Services   int
	Containers int
	Running    int
	Exited     int
	// ConfigFiles are the compose files that define the project, from the
	// com.docker.compose.project.config_files label or a directory scan.
	ConfigFiles []string
	// WorkingDir is the directory compose resolves relative paths against.
	WorkingDir string
	Status     ProjectStatus
}

// ComposeNetwork represents a network created by Docker Compose.
type ComposeNetwork struct {
	Name   string
	Driver string
	Scope  string
}

// ComposeVolume represents a volume created by Docker Compose.
type ComposeVolume struct {
	Name   string
	Driver string
}

// ComposeService represents a service within a Docker Compose project.
type ComposeService struct {
	Project    string
	Name       string
	Containers int
	Running    int
	Exited     int
	Image      string
	Health     string // "healthy", "unhealthy", "starting", "none", or ""
	Ports      string // formatted listening ports
}

// ProjectWithServices pairs a project with its services.
type ProjectWithServices struct {
	Project  ComposeProject
	Services []ComposeService
}

// AggregateComposeAll produces projects with their services embedded in a single pass.
func AggregateComposeAll(containers []*Container) []ProjectWithServices {
	projects := AggregateComposeProjects(containers)
	result := make([]ProjectWithServices, len(projects))
	for i, p := range projects {
		result[i] = ProjectWithServices{
			Project:  p,
			Services: AggregateComposeServices(containers, p.Name),
		}
	}
	return result
}

// AggregateComposeProjects groups containers by their com.docker.compose.project label.
func AggregateComposeProjects(containers []*Container) []ComposeProject {
	type projectAcc struct {
		services    map[string]bool
		containers  int
		running     int
		exited      int
		configFiles []string
		workingDir  string
	}
	projects := make(map[string]*projectAcc)
	for _, c := range containers {
		project := c.Labels["com.docker.compose.project"]
		service := c.Labels["com.docker.compose.service"]
		if project == "" || service == "" {
			continue
		}
		if c.Labels["com.docker.compose.oneoff"] == "True" {
			continue
		}
		acc, ok := projects[project]
		if !ok {
			acc = &projectAcc{services: make(map[string]bool)}
			projects[project] = acc
		}
		acc.services[service] = true
		acc.containers++
		if IsContainerRunning(c) {
			acc.running++
		} else {
			acc.exited++
		}
		if len(acc.configFiles) == 0 {
			if raw := c.Labels["com.docker.compose.project.config_files"]; raw != "" {
				for _, f := range strings.Split(raw, ",") {
					if f = strings.TrimSpace(f); f != "" {
						acc.configFiles = append(acc.configFiles, f)
					}
				}
			}
		}
		if acc.workingDir == "" {
			acc.workingDir = c.Labels["com.docker.compose.project.working_dir"]
		}
	}

	result := make([]ComposeProject, 0, len(projects))
	for name, acc := range projects {
		status := ProjectStopped
		if acc.running > 0 {
			status = ProjectRunning
		}
		result = append(result, ComposeProject{
			Name:        name,
			Services:    len(acc.services),
			Containers:  acc.containers,
			Running:     acc.running,
			Exited:      acc.exited,
			ConfigFiles: acc.configFiles,
			WorkingDir:  acc.workingDir,
			Status:      status,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// AggregateComposeServices groups containers for a specific project by their
// com.docker.compose.service label.
func AggregateComposeServices(containers []*Container, project string) []ComposeService {
	type serviceAcc struct {
		containers int
		running    int
		exited     int
		image      string
		healths    []string
		ports      []container.PortSummary
	}
	services := make(map[string]*serviceAcc)
	for _, c := range containers {
		if c.Labels["com.docker.compose.project"] != project {
			continue
		}
		svc := c.Labels["com.docker.compose.service"]
		if svc == "" {
			continue
		}
		if c.Labels["com.docker.compose.oneoff"] == "True" {
			continue
		}
		acc, ok := services[svc]
		if !ok {
			acc = &serviceAcc{}
			services[svc] = acc
		}
		acc.containers++
		if IsContainerRunning(c) {
			acc.running++
		} else {
			acc.exited++
		}
		if acc.image == "" && c.Image != "" {
			acc.image = c.Image
		}
		health := ""
		if c.Detail.State != nil && c.Detail.State.Health != nil {
			health = string(c.Detail.State.Health.Status)
		}
		acc.healths = append(acc.healths, health)
		acc.ports = append(acc.ports, c.Ports...)
	}

	result := make([]ComposeService, 0, len(services))
	for name, acc := range services {
		result = append(result, ComposeService{
			Project:    project,
			Name:       name,
			Containers: acc.containers,
			Running:    acc.running,
			Exited:     acc.exited,
			Image:      acc.image,
			Health:     aggregateHealth(acc.healths),
			Ports:      aggregatePorts(acc.ports),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// ServiceSync says whether a service's running containers match the compose
// file that defines them.
type ServiceSync string

const (
	// ServiceInSync means every container matches the file.
	ServiceInSync ServiceSync = "in sync"
	// ServiceDrifted means the file changed since a container was created,
	// so the next `up` will recreate it.
	ServiceDrifted ServiceSync = "drifted"
	// ServiceNotCreated means the file defines the service but nothing runs it.
	ServiceNotCreated ServiceSync = "not created"
	// ServiceUnknown means the comparison could not be made, usually because
	// the project's compose files are not known.
	ServiceUnknown ServiceSync = ""
)

// CompareConfigHashes compares each service's container config-hash labels
// against the hashes compose computes for the project's files. This is the
// same test compose itself applies when deciding whether to recreate a
// container. With no file hashes, every existing service is unknown rather
// than falsely in sync.
func CompareConfigHashes(containers []*Container, project string, fileHashes map[string]string) map[string]ServiceSync {
	status := make(map[string]ServiceSync)
	for _, c := range containers {
		if c.Labels["com.docker.compose.project"] != project {
			continue
		}
		if c.Labels["com.docker.compose.oneoff"] == "True" {
			continue
		}
		service := c.Labels["com.docker.compose.service"]
		if service == "" {
			continue
		}
		if len(fileHashes) == 0 {
			status[service] = ServiceUnknown
			continue
		}
		want, known := fileHashes[service]
		if !known {
			// The file no longer defines this service; leave it unknown
			// rather than claim drift we cannot substantiate.
			status[service] = ServiceUnknown
			continue
		}
		if status[service] == ServiceDrifted {
			continue
		}
		if c.Labels["com.docker.compose.config-hash"] == want {
			status[service] = ServiceInSync
		} else {
			status[service] = ServiceDrifted
		}
	}
	for service := range fileHashes {
		if _, seen := status[service]; !seen {
			status[service] = ServiceNotCreated
		}
	}
	return status
}

// aggregateHealth derives a single health status from individual container health statuses.
// If any unhealthy -> "unhealthy", else if any starting -> "starting",
// else if all healthy -> "healthy", else "none".
func aggregateHealth(healths []string) string {
	hasHealthy := false
	for _, h := range healths {
		switch h {
		case "unhealthy":
			return "unhealthy"
		case "healthy":
			hasHealthy = true
		}
	}
	for _, h := range healths {
		if h == "starting" {
			return "starting"
		}
	}
	if hasHealthy {
		return "healthy"
	}
	return "none"
}

// aggregatePorts deduplicates and formats ports from all containers in a service.
func aggregatePorts(ports []container.PortSummary) string {
	if len(ports) == 0 {
		return ""
	}
	// Deduplicate by (IP, PublicPort, PrivatePort, Type) tuple.
	type portKey struct {
		IP          string
		PublicPort  uint16
		PrivatePort uint16
		Type        string
	}
	seen := make(map[portKey]bool)
	var unique []container.PortSummary
	for _, p := range ports {
		var ipAddr string
		if p.IP.IsValid() {
			ipAddr = p.IP.String()
		}
		k := portKey{ipAddr, p.PublicPort, p.PrivatePort, p.Type}
		if !seen[k] {
			seen[k] = true
			unique = append(unique, p)
		}
	}
	// Sort by private port for consistent display.
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].PrivatePort < unique[j].PrivatePort
	})
	var parts []string
	for _, p := range unique {
		if p.PublicPort != 0 {
			parts = append(parts, fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type))
		} else {
			parts = append(parts, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
		}
	}
	return strings.Join(parts, ", ")
}
