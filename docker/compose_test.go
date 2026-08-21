package docker

import (
	"testing"

	"github.com/moby/moby/api/types/container"
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

func TestAggregateComposeProjects_CarriesFilesAndStatus(t *testing.T) {
	containers := []*Container{
		composeTestContainer("web-1", map[string]string{
			"com.docker.compose.project":              "web",
			"com.docker.compose.service":              "api",
			"com.docker.compose.project.config_files": "/srv/web/compose.yaml,/srv/web/override.yaml",
			"com.docker.compose.project.working_dir":  "/srv/web",
		}, true),
		composeTestContainer("idle-1", map[string]string{
			"com.docker.compose.project":              "idle",
			"com.docker.compose.service":              "worker",
			"com.docker.compose.project.config_files": "/srv/idle/compose.yaml",
			"com.docker.compose.project.working_dir":  "/srv/idle",
		}, false),
	}

	projects := AggregateComposeProjects(containers)
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	byName := map[string]ComposeProject{}
	for _, p := range projects {
		byName[p.Name] = p
	}

	web := byName["web"]
	if len(web.ConfigFiles) != 2 || web.ConfigFiles[1] != "/srv/web/override.yaml" {
		t.Fatalf("expected the comma-separated files to be split, got %v", web.ConfigFiles)
	}
	if web.WorkingDir != "/srv/web" {
		t.Fatalf("expected the working dir label, got %q", web.WorkingDir)
	}
	if web.Status != ProjectRunning {
		t.Fatalf("expected a project with a running container to be running, got %q", web.Status)
	}
	if byName["idle"].Status != ProjectStopped {
		t.Fatalf("expected a project with only exited containers to be stopped, got %q", byName["idle"].Status)
	}
}

// composeTestContainer builds a container with the given name, labels, and
// running state. IsContainerRunning tests whether Status contains "Up", so
// the running flag must be expressed through Status, not State.
func composeTestContainer(name string, labels map[string]string, running bool) *Container {
	status := "Exited (0) 1 hour ago"
	if running {
		status = "Up 2 minutes"
	}
	return &Container{
		Summary: container.Summary{
			ID:     name,
			Names:  []string{"/" + name},
			Labels: labels,
			Status: status,
		},
	}
}

func TestCompareConfigHashes(t *testing.T) {
	containers := []*Container{
		composeTestContainer("web-api-1", map[string]string{
			"com.docker.compose.project":     "web",
			"com.docker.compose.service":     "api",
			"com.docker.compose.config-hash": "aaa",
		}, true),
		composeTestContainer("web-old-1", map[string]string{
			"com.docker.compose.project":     "web",
			"com.docker.compose.service":     "old",
			"com.docker.compose.config-hash": "stale",
		}, true),
		composeTestContainer("other-1", map[string]string{
			"com.docker.compose.project":     "other",
			"com.docker.compose.service":     "api",
			"com.docker.compose.config-hash": "zzz",
		}, true),
	}
	fileHashes := map[string]string{"api": "aaa", "old": "fresh", "brandnew": "bbb"}

	got := CompareConfigHashes(containers, "web", fileHashes)

	if got["api"] != ServiceInSync {
		t.Errorf("api: expected in sync, got %q", got["api"])
	}
	if got["old"] != ServiceDrifted {
		t.Errorf("old: expected drifted, got %q", got["old"])
	}
	if got["brandnew"] != ServiceNotCreated {
		t.Errorf("brandnew: expected not created, got %q", got["brandnew"])
	}
	if _, ok := got["other"]; ok {
		t.Error("expected another project's services to be ignored")
	}
}

func TestCompareConfigHashes_NoFileHashesMeansUnknown(t *testing.T) {
	containers := []*Container{
		composeTestContainer("web-api-1", map[string]string{
			"com.docker.compose.project":     "web",
			"com.docker.compose.service":     "api",
			"com.docker.compose.config-hash": "aaa",
		}, true),
	}

	got := CompareConfigHashes(containers, "web", nil)
	if got["api"] != ServiceUnknown {
		t.Fatalf("expected unknown without file hashes, got %q", got["api"])
	}
}

// TestCompareConfigHashes_DriftStickyAcrossContainers guards the
// `if status[service] == ServiceDrifted { continue }` check. The two
// containers are ordered so a stale one is seen first and a matching one
// second; without the guard, the second (in-sync) container would overwrite
// the drift already observed from the first, silently hiding it. A scaled
// service with even one stale container has drifted, regardless of order.
func TestCompareConfigHashes_DriftStickyAcrossContainers(t *testing.T) {
	containers := []*Container{
		composeTestContainer("web-api-1", map[string]string{
			"com.docker.compose.project":     "web",
			"com.docker.compose.service":     "api",
			"com.docker.compose.config-hash": "stale",
		}, true),
		composeTestContainer("web-api-2", map[string]string{
			"com.docker.compose.project":     "web",
			"com.docker.compose.service":     "api",
			"com.docker.compose.config-hash": "aaa",
		}, true),
	}
	fileHashes := map[string]string{"api": "aaa"}

	got := CompareConfigHashes(containers, "web", fileHashes)
	if got["api"] != ServiceDrifted {
		t.Fatalf("expected drift from the stale container to stick, got %q", got["api"])
	}
}

// TestCompareConfigHashes_OneOffContainerIgnored guards the
// `com.docker.compose.oneoff == "True"` check. The only container for this
// service is a one-off (e.g. a `compose run`), so the service itself has no
// real container running it; without the guard, the one-off's mismatched
// hash would be read as drift instead of "not created".
func TestCompareConfigHashes_OneOffContainerIgnored(t *testing.T) {
	containers := []*Container{
		composeTestContainer("web-api-run", map[string]string{
			"com.docker.compose.project":     "web",
			"com.docker.compose.service":     "api",
			"com.docker.compose.config-hash": "mismatch",
			"com.docker.compose.oneoff":      "True",
		}, true),
	}
	fileHashes := map[string]string{"api": "aaa"}

	got := CompareConfigHashes(containers, "web", fileHashes)
	if got["api"] != ServiceNotCreated {
		t.Fatalf("expected the one-off container to be ignored entirely, got %q", got["api"])
	}
}

// TestCompareConfigHashes_ServiceNotInFileHashesIsUnknown guards the
// `!known` branch, reached only when fileHashes is non-empty overall but
// does not define this particular service (e.g. the file was edited to
// remove it). Without the guard, the zero-value "" from the map lookup
// would be compared against the container's real hash and almost always
// read as drift, misreporting a service compose no longer even defines.
func TestCompareConfigHashes_ServiceNotInFileHashesIsUnknown(t *testing.T) {
	containers := []*Container{
		composeTestContainer("web-orphan-1", map[string]string{
			"com.docker.compose.project":     "web",
			"com.docker.compose.service":     "orphan",
			"com.docker.compose.config-hash": "whatever",
		}, true),
	}
	fileHashes := map[string]string{"other": "x"}

	got := CompareConfigHashes(containers, "web", fileHashes)
	if got["orphan"] != ServiceUnknown {
		t.Fatalf("expected a service missing from fileHashes to be unknown, got %q", got["orphan"])
	}
}
