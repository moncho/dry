package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/api/types/container"
	dockerAPI "github.com/docker/docker/client"
)

// composeTestClient is a mock Docker API client for compose action tests.
type composeTestClient struct {
	dockerAPI.APIClient
	startErr   map[string]error
	stopErr    map[string]error
	restartErr map[string]error
	removeErr  map[string]error
}

func (c *composeTestClient) ContainerStart(ctx context.Context, id string, opts container.StartOptions) error {
	if c.startErr != nil {
		if err, ok := c.startErr[id]; ok {
			return err
		}
	}
	return nil
}

func (c *composeTestClient) ContainerStop(ctx context.Context, id string, opts container.StopOptions) error {
	if c.stopErr != nil {
		if err, ok := c.stopErr[id]; ok {
			return err
		}
	}
	return nil
}

func (c *composeTestClient) ContainerRestart(ctx context.Context, id string, opts container.StopOptions) error {
	if c.restartErr != nil {
		if err, ok := c.restartErr[id]; ok {
			return err
		}
	}
	return nil
}

func (c *composeTestClient) ContainerRemove(ctx context.Context, id string, opts container.RemoveOptions) error {
	if c.removeErr != nil {
		if err, ok := c.removeErr[id]; ok {
			return err
		}
	}
	return nil
}

// ContainerList is needed for refreshAndWait → NewDockerContainerStore.
func (c *composeTestClient) ContainerList(ctx context.Context, opts container.ListOptions) ([]container.Summary, error) {
	return nil, nil
}

// simpleStore implements ContainerStore for test pre-population.
type simpleStore struct {
	containers []*Container
}

func (s *simpleStore) Get(id string) *Container {
	for _, c := range s.containers {
		if c.ID == id {
			return c
		}
	}
	return nil
}
func (s *simpleStore) List() []*Container       { return s.containers }
func (s *simpleStore) Remove(id string)         {}
func (s *simpleStore) Size() int                { return len(s.containers) }

func newTestDaemon(containers []*Container, client *composeTestClient) *DockerDaemon {
	return &DockerDaemon{
		client: client,
		s:      &simpleStore{containers: containers},
	}
}

func TestComposeServiceContainers_MatchesByLabels(t *testing.T) {
	containers := []*Container{
		makeContainer("aaa111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("bbb111222333", "Exited (0)", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "db",
		}, "pg:15"),
		makeContainer("ccc111222333", "Up 2h", map[string]string{
			"com.docker.compose.project": "other",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		// One-off excluded
		makeContainer("ddd111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
			"com.docker.compose.oneoff":  "True",
		}, "api:latest"),
	}
	daemon := newTestDaemon(containers, &composeTestClient{})
	matched := composeServiceContainers(daemon, "web", "api")
	if len(matched) != 1 {
		t.Fatalf("expected 1 container, got %d", len(matched))
	}
	if matched[0].ID != "aaa111222333" {
		t.Errorf("expected container aaa111222333, got %s", matched[0].ID)
	}
}

func TestComposeServiceStart_SkipsRunning(t *testing.T) {
	containers := []*Container{
		makeContainer("run111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("stp111222333", "Exited (0)", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
	}
	daemon := newTestDaemon(containers, &composeTestClient{})
	report, err := daemon.ComposeServiceStart("web", "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Targeted != 2 {
		t.Errorf("expected 2 targeted, got %d", report.Targeted)
	}
	if report.Skipped != 1 {
		t.Errorf("expected 1 skipped (running), got %d", report.Skipped)
	}
	if report.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", report.Succeeded)
	}
	if report.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", report.Failed)
	}
}

func TestComposeServiceStop_SkipsStopped(t *testing.T) {
	containers := []*Container{
		makeContainer("run111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("stp111222333", "Exited (0)", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
	}
	daemon := newTestDaemon(containers, &composeTestClient{})
	report, err := daemon.ComposeServiceStop("web", "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Targeted != 2 {
		t.Errorf("expected 2 targeted, got %d", report.Targeted)
	}
	if report.Skipped != 1 {
		t.Errorf("expected 1 skipped (stopped), got %d", report.Skipped)
	}
	if report.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", report.Succeeded)
	}
}

func TestComposeServiceRestart_AllContainers(t *testing.T) {
	containers := []*Container{
		makeContainer("aaa111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("bbb111222333", "Exited (0)", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
	}
	daemon := newTestDaemon(containers, &composeTestClient{})
	report, err := daemon.ComposeServiceRestart("web", "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Targeted != 2 {
		t.Errorf("expected 2 targeted, got %d", report.Targeted)
	}
	if report.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", report.Skipped)
	}
	if report.Succeeded != 2 {
		t.Errorf("expected 2 succeeded, got %d", report.Succeeded)
	}
}

func TestComposeServiceRemove_AllContainers(t *testing.T) {
	containers := []*Container{
		makeContainer("aaa111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("bbb111222333", "Exited (0)", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
	}
	daemon := newTestDaemon(containers, &composeTestClient{})
	report, err := daemon.ComposeServiceRemove("web", "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Targeted != 2 {
		t.Errorf("expected 2 targeted, got %d", report.Targeted)
	}
	if report.Succeeded != 2 {
		t.Errorf("expected 2 succeeded, got %d", report.Succeeded)
	}
	if report.Action != "remove" {
		t.Errorf("expected action 'remove', got %q", report.Action)
	}
}

func TestComposeServiceStart_PartialFailure(t *testing.T) {
	containers := []*Container{
		makeContainer("ok1111222333", "Exited (0)", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("fail11222333", "Exited (1)", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
	}
	client := &composeTestClient{
		startErr: map[string]error{
			"fail11222333": errors.New("cannot start"),
		},
	}
	daemon := newTestDaemon(containers, client)
	report, err := daemon.ComposeServiceStart("web", "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", report.Succeeded)
	}
	if report.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", report.Failed)
	}
	if len(report.Errors) != 1 {
		t.Errorf("expected 1 error message, got %d", len(report.Errors))
	}
}

func TestComposeServiceActionReport_Summary(t *testing.T) {
	report := ComposeServiceActionReport{
		Project:   "web",
		Service:   "api",
		Action:    "stop",
		Targeted:  3,
		Succeeded: 2,
		Skipped:   1,
		Failed:    0,
	}
	summary := report.Summary()
	if summary != "Stop web/api: 3 targeted, 2 succeeded, 1 skipped" {
		t.Errorf("unexpected summary: %q", summary)
	}

	report.Failed = 1
	summary = report.Summary()
	if summary != "Stop web/api: 3 targeted, 2 succeeded, 1 skipped, 1 failed" {
		t.Errorf("unexpected summary with failures: %q", summary)
	}
}

func TestComposeServiceContainers_DeterministicOrder(t *testing.T) {
	containers := []*Container{
		{Summary: container.Summary{
			ID:     "zzz111222333",
			Names:  []string{"/web-api-2"},
			Status: "Up 1h",
			Labels: map[string]string{
				"com.docker.compose.project": "web",
				"com.docker.compose.service": "api",
			},
		}},
		{Summary: container.Summary{
			ID:     "aaa111222333",
			Names:  []string{"/web-api-1"},
			Status: "Up 1h",
			Labels: map[string]string{
				"com.docker.compose.project": "web",
				"com.docker.compose.service": "api",
			},
		}},
	}
	daemon := newTestDaemon(containers, &composeTestClient{})
	matched := composeServiceContainers(daemon, "web", "api")
	if len(matched) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(matched))
	}
	// Sorted by name: /web-api-1 before /web-api-2
	if matched[0].ID != "aaa111222333" {
		t.Errorf("expected first container aaa111222333, got %s", matched[0].ID)
	}
	if matched[1].ID != "zzz111222333" {
		t.Errorf("expected second container zzz111222333, got %s", matched[1].ID)
	}
}

// --- Project-level action tests ---

func TestComposeProjectContainers_MatchesAllServices(t *testing.T) {
	containers := []*Container{
		makeContainer("aaa111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("bbb111222333", "Up 2h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "db",
		}, "pg:15"),
		makeContainer("ccc111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "other",
			"com.docker.compose.service": "svc",
		}, "svc:latest"),
		// One-off excluded
		makeContainer("ddd111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
			"com.docker.compose.oneoff":  "True",
		}, "api:latest"),
	}
	daemon := newTestDaemon(containers, &composeTestClient{})
	matched := composeProjectContainers(daemon, "web")
	if len(matched) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(matched))
	}
}

func TestComposeProjectStop_SkipsStopped(t *testing.T) {
	containers := []*Container{
		makeContainer("run111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("stp111222333", "Exited (0)", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "db",
		}, "pg:15"),
	}
	daemon := newTestDaemon(containers, &composeTestClient{})
	report, err := daemon.ComposeProjectStop("web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Targeted != 2 {
		t.Errorf("expected 2 targeted, got %d", report.Targeted)
	}
	if report.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", report.Skipped)
	}
	if report.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", report.Succeeded)
	}
	if report.Service != "" {
		t.Errorf("expected empty service, got %q", report.Service)
	}
}

func TestComposeProjectRestart_AllContainers(t *testing.T) {
	containers := []*Container{
		makeContainer("aaa111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("bbb111222333", "Exited (0)", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "db",
		}, "pg:15"),
	}
	daemon := newTestDaemon(containers, &composeTestClient{})
	report, err := daemon.ComposeProjectRestart("web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Targeted != 2 {
		t.Errorf("expected 2 targeted, got %d", report.Targeted)
	}
	if report.Succeeded != 2 {
		t.Errorf("expected 2 succeeded, got %d", report.Succeeded)
	}
}

func TestComposeProjectRemove_AllContainers(t *testing.T) {
	containers := []*Container{
		makeContainer("aaa111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("bbb111222333", "Up 2h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "db",
		}, "pg:15"),
	}
	daemon := newTestDaemon(containers, &composeTestClient{})
	report, err := daemon.ComposeProjectRemove("web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Targeted != 2 {
		t.Errorf("expected 2 targeted, got %d", report.Targeted)
	}
	if report.Succeeded != 2 {
		t.Errorf("expected 2 succeeded, got %d", report.Succeeded)
	}
	if report.Action != "remove" {
		t.Errorf("expected action 'remove', got %q", report.Action)
	}
}

func TestComposeProjectStop_PartialFailure(t *testing.T) {
	containers := []*Container{
		makeContainer("ok1111222333", "Up 1h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "api",
		}, "api:latest"),
		makeContainer("fail11222333", "Up 2h", map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "db",
		}, "pg:15"),
	}
	client := &composeTestClient{
		stopErr: map[string]error{
			"fail11222333": errors.New("cannot stop"),
		},
	}
	daemon := newTestDaemon(containers, client)
	report, err := daemon.ComposeProjectStop("web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", report.Succeeded)
	}
	if report.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", report.Failed)
	}
	if len(report.Errors) != 1 {
		t.Errorf("expected 1 error message, got %d", len(report.Errors))
	}
}

func TestComposeProjectActionReport_Summary(t *testing.T) {
	report := ComposeServiceActionReport{
		Project:   "web",
		Action:    "stop",
		Targeted:  4,
		Succeeded: 3,
		Skipped:   1,
	}
	summary := report.Summary()
	if summary != "Stop web: 4 targeted, 3 succeeded, 1 skipped" {
		t.Errorf("unexpected project-level summary: %q", summary)
	}
}
