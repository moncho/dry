package compose

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

// TestProjectsModel_SetDrift_RendersSyncColumn proves a drifted service
// actually renders as drifted, and an in-sync one as in sync, end to end
// through SetDrift and the row's Columns(). The regenerated golden
// snapshots always render the SYNC column empty (nothing drives SetDrift
// there), so without this nothing anywhere asserted that a drifted service
// visibly looks different from one that is not.
func TestProjectsModel_SetDrift_RendersSyncColumn(t *testing.T) {
	m := NewProjectsModel()
	m.SetSize(120, 40)
	m.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "web"},
		Services: []docker.ComposeService{
			{Project: "web", Name: "api"},
			{Project: "web", Name: "worker"},
		},
	}})
	m.SetDrift(map[string]map[string]docker.ServiceSync{
		"web": {
			"api":    docker.ServiceDrifted,
			"worker": docker.ServiceInSync,
		},
	})

	var apiCols, workerCols []string
	for _, row := range m.table.FilteredRows() {
		r, ok := row.(serviceDetailRow)
		if !ok {
			continue
		}
		switch r.service.Name {
		case "api":
			apiCols = r.Columns()
		case "worker":
			workerCols = r.Columns()
		}
	}
	if apiCols == nil || workerCols == nil {
		t.Fatalf("expected both service rows to be present, got api=%v worker=%v", apiCols, workerCols)
	}

	const syncCol = 6 // NAME, CONTAINERS, RUNNING, EXITED, STATUS/IMAGE, HEALTH, SYNC, PORTS
	if !strings.Contains(apiCols[syncCol], "drift") {
		t.Fatalf("expected the drifted service's SYNC cell to render \"drift\", got %q", apiCols[syncCol])
	}
	if !strings.Contains(workerCols[syncCol], "ok") {
		t.Fatalf("expected the in-sync service's SYNC cell to render \"ok\", got %q", workerCols[syncCol])
	}
	if strings.Contains(apiCols[syncCol], "ok") {
		t.Fatalf("expected the drifted service's SYNC cell not to also read ok, got %q", apiCols[syncCol])
	}
}

// TestProjectsModel_ProjectHeaderRendersStatus is the spec's stated
// observable outcome: the project's lifecycle status on screen, so a
// file-discovered project that is down reads as "not created" instead of as
// a row of blanks indistinguishable from noise. It also pins the two
// columns' meanings apart: STATUS is per-project lifecycle and lives on the
// project header row, SYNC is per-service drift and stays empty there.
func TestProjectsModel_ProjectHeaderRendersStatus(t *testing.T) {
	m := NewProjectsModel()
	m.SetSize(120, 40)
	m.SetProjects([]docker.ProjectWithServices{
		{Project: docker.ComposeProject{Name: "up", Running: 1, Status: docker.ProjectRunning}},
		{Project: docker.ComposeProject{Name: "down", Exited: 1, Status: docker.ProjectStopped}},
		{Project: docker.ComposeProject{Name: "fileonly", Status: docker.ProjectNotCreated}},
		{Project: docker.ComposeProject{Name: "unknown"}},
	})

	// Cell indexes: NAME, CONTAINERS, RUNNING, EXITED, STATUS/IMAGE,
	// HEALTH, SYNC, PORTS.
	const (
		statusCol = 4
		syncCol   = 6
	)
	want := map[string]string{
		"up":       "running",
		"down":     "stopped",
		"fileonly": "not created",
		"unknown":  "", // no status computed: render nothing, invent nothing
	}
	seen := 0
	for _, row := range m.table.FilteredRows() {
		r, ok := row.(projectHeaderRow)
		if !ok {
			continue
		}
		seen++
		cols := r.Columns()
		expected, known := want[r.project.Name]
		if !known {
			t.Fatalf("unexpected project row %q", r.project.Name)
		}
		if got := ansi.Strip(cols[statusCol]); got != expected {
			t.Fatalf("project %s: expected STATUS cell %q, got %q", r.project.Name, expected, got)
		}
		if got := cols[syncCol]; got != "" {
			t.Fatalf("project %s: SYNC is per-service drift and must stay empty on a project row, got %q", r.project.Name, got)
		}
	}
	if seen != len(want) {
		t.Fatalf("expected %d project header rows, saw %d", len(want), seen)
	}
}

// TestProjectsModel_StatusReachesTheScreen renders the view the way the app
// does and reads it back: the status must appear under a column header that
// names it, and it must not cost a neighbouring column its data — a previous
// round of this feature shipped a new column that silently truncated PORTS.
func TestProjectsModel_StatusReachesTheScreen(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(120, 40)
	m.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{
			Name: "web", Services: 1, Containers: 1, Running: 1,
			Status: docker.ProjectRunning,
		},
		Services: []docker.ComposeService{{
			Project: "web", Name: "api", Containers: 1, Running: 1,
			Image: "api:latest", Health: "healthy", Ports: "0.0.0.0:8080->8080/tcp",
		}},
	}})

	lines := strings.Split(ansi.Strip(m.View()), "\n")
	var header, projectRow, serviceRow string
	for _, l := range lines {
		switch {
		case strings.Contains(l, "CONTAINERS"):
			header = l
		case strings.Contains(l, "web"):
			projectRow = l
		case strings.Contains(l, "api:latest"):
			serviceRow = l
		}
	}
	if header == "" || projectRow == "" || serviceRow == "" {
		t.Fatalf("expected a header, a project row and a service row, got:\n%s", strings.Join(lines, "\n"))
	}
	if !strings.Contains(header, "STATUS") {
		t.Fatalf("expected the column header to name STATUS, got %q", header)
	}
	if !strings.Contains(projectRow, "running") {
		t.Fatalf("expected the project row to show its status, got %q", projectRow)
	}
	// Data the status must not have displaced.
	if !strings.Contains(serviceRow, "0.0.0.0:8080->8080/tcp") {
		t.Fatalf("PORTS lost data: %q", serviceRow)
	}
	if !strings.Contains(serviceRow, "api:latest") || !strings.Contains(serviceRow, "healthy") {
		t.Fatalf("IMAGE or HEALTH lost data: %q", serviceRow)
	}
	if !strings.Contains(header, "SYNC") || !strings.Contains(header, "PORTS") {
		t.Fatalf("a column header went missing: %q", header)
	}
}

// TestProjectsModel_SetDrift_MissingEntryRendersEmpty covers a service with
// no drift entry at all (e.g. drift has not been computed yet, or the
// project's files are unknown): its SYNC cell must stay empty rather than
// default to any of the rendered labels.
func TestProjectsModel_SetDrift_MissingEntryRendersEmpty(t *testing.T) {
	m := NewProjectsModel()
	m.SetSize(120, 40)
	m.SetProjects([]docker.ProjectWithServices{{
		Project:  docker.ComposeProject{Name: "web"},
		Services: []docker.ComposeService{{Project: "web", Name: "api"}},
	}})

	for _, row := range m.table.FilteredRows() {
		r, ok := row.(serviceDetailRow)
		if !ok {
			continue
		}
		if got := r.Columns()[6]; got != "" {
			t.Fatalf("expected an empty SYNC cell with no drift set, got %q", got)
		}
	}
}
