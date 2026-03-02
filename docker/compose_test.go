package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestAggregateComposeProjects(t *testing.T) {
	tests := []struct {
		name       string
		containers []*Container
		wantCount  int
		check      func(t *testing.T, projects []ComposeProject)
	}{
		{
			name:       "empty input",
			containers: nil,
			wantCount:  0,
		},
		{
			name: "missing labels skipped",
			containers: []*Container{
				makeContainer("c1", "Up 1h", map[string]string{}, "img:latest"),
			},
			wantCount: 0,
		},
		{
			name: "one-off containers excluded",
			containers: []*Container{
				makeContainer("c1", "Up 1h", map[string]string{
					"com.docker.compose.project": "web",
					"com.docker.compose.service": "api",
					"com.docker.compose.oneoff":  "True",
				}, "img:latest"),
			},
			wantCount: 0,
		},
		{
			name: "mixed running and exited",
			containers: []*Container{
				makeContainer("c1", "Up 1h", map[string]string{
					"com.docker.compose.project": "web",
					"com.docker.compose.service": "api",
				}, "api:latest"),
				makeContainer("c2", "Exited (0) 5m", map[string]string{
					"com.docker.compose.project": "web",
					"com.docker.compose.service": "worker",
				}, "worker:latest"),
				makeContainer("c3", "Up 2h", map[string]string{
					"com.docker.compose.project": "web",
					"com.docker.compose.service": "api",
				}, "api:latest"),
			},
			wantCount: 1,
			check: func(t *testing.T, projects []ComposeProject) {
				p := projects[0]
				if p.Name != "web" {
					t.Errorf("expected project name 'web', got %q", p.Name)
				}
				if p.Services != 2 {
					t.Errorf("expected 2 services, got %d", p.Services)
				}
				if p.Containers != 3 {
					t.Errorf("expected 3 containers, got %d", p.Containers)
				}
				if p.Running != 2 {
					t.Errorf("expected 2 running, got %d", p.Running)
				}
				if p.Exited != 1 {
					t.Errorf("expected 1 exited, got %d", p.Exited)
				}
			},
		},
		{
			name: "multiple projects sorted by name",
			containers: []*Container{
				makeContainer("c1", "Up 1h", map[string]string{
					"com.docker.compose.project": "zoo",
					"com.docker.compose.service": "app",
				}, "zoo:latest"),
				makeContainer("c2", "Up 1h", map[string]string{
					"com.docker.compose.project": "alpha",
					"com.docker.compose.service": "web",
				}, "alpha:latest"),
			},
			wantCount: 2,
			check: func(t *testing.T, projects []ComposeProject) {
				if projects[0].Name != "alpha" {
					t.Errorf("expected first project 'alpha', got %q", projects[0].Name)
				}
				if projects[1].Name != "zoo" {
					t.Errorf("expected second project 'zoo', got %q", projects[1].Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects := AggregateComposeProjects(tt.containers)
			if len(projects) != tt.wantCount {
				t.Fatalf("expected %d projects, got %d", tt.wantCount, len(projects))
			}
			if tt.check != nil {
				tt.check(t, projects)
			}
		})
	}
}

func TestAggregateComposeServices(t *testing.T) {
	containers := []*Container{
		makeContainer("c1", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("c2", "Exited (0) 5m", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("c3", "Up 2h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "db",
		}, "postgres:15"),
		// Different project — should be excluded
		makeContainer("c4", "Up 1h", map[string]string{
			"com.docker.compose.project": "other",
			"com.docker.compose.service": "svc",
		}, "other:latest"),
		// One-off — should be excluded
		makeContainer("c5", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
			"com.docker.compose.oneoff":  "True",
		}, "api:latest"),
	}

	services := AggregateComposeServices(containers, "web")
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}

	// Sorted by name: api, db
	api := services[0]
	if api.Name != "api" {
		t.Errorf("expected service 'api', got %q", api.Name)
	}
	if api.Containers != 2 {
		t.Errorf("expected 2 containers for api, got %d", api.Containers)
	}
	if api.Running != 1 {
		t.Errorf("expected 1 running for api, got %d", api.Running)
	}
	if api.Exited != 1 {
		t.Errorf("expected 1 exited for api, got %d", api.Exited)
	}
	if api.Image != "api:latest" {
		t.Errorf("expected image 'api:latest', got %q", api.Image)
	}

	db := services[1]
	if db.Name != "db" {
		t.Errorf("expected service 'db', got %q", db.Name)
	}
	if db.Image != "postgres:15" {
		t.Errorf("expected image 'postgres:15', got %q", db.Image)
	}
}

func TestAggregateHealth(t *testing.T) {
	tests := []struct {
		name    string
		healths []string
		want    string
	}{
		{"empty", nil, "none"},
		{"all healthy", []string{"healthy", "healthy"}, "healthy"},
		{"any unhealthy", []string{"healthy", "unhealthy"}, "unhealthy"},
		{"any starting", []string{"healthy", "starting"}, "starting"},
		{"unhealthy takes priority over starting", []string{"unhealthy", "starting"}, "unhealthy"},
		{"no health info", []string{"", ""}, "none"},
		{"mixed healthy and none", []string{"healthy", ""}, "healthy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregateHealth(tt.healths)
			if got != tt.want {
				t.Errorf("aggregateHealth(%v) = %q, want %q", tt.healths, got, tt.want)
			}
		})
	}
}

func makeContainer(id, status string, labels map[string]string, image string) *Container {
	return &Container{
		Summary: container.Summary{
			ID:     id,
			Status: status,
			Labels: labels,
			Image:  image,
		},
	}
}
