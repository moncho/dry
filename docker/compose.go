package docker

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docker/docker/api/types/container"
)

// ComposeProject represents a Docker Compose project aggregated from container labels.
type ComposeProject struct {
	Name       string
	Services   int
	Containers int
	Running    int
	Exited     int
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
		services   map[string]bool
		containers int
		running    int
		exited     int
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
	}

	result := make([]ComposeProject, 0, len(projects))
	for name, acc := range projects {
		result = append(result, ComposeProject{
			Name:       name,
			Services:   len(acc.services),
			Containers: acc.containers,
			Running:    acc.running,
			Exited:     acc.exited,
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
		ports      []container.Port
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
		if c.Detail.ContainerJSONBase != nil && c.Detail.State != nil && c.Detail.State.Health != nil {
			health = c.Detail.State.Health.Status
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
func aggregatePorts(ports []container.Port) string {
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
	var unique []container.Port
	for _, p := range ports {
		k := portKey{p.IP, p.PublicPort, p.PrivatePort, p.Type}
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
