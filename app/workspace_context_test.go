package app

import (
	"strings"
	"testing"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

// The workspace panel sits beside the project list, so its service count has
// to mean the same thing the list shows. p.Services is counted from
// containers and cannot see the services the compose file defines and
// nothing runs, which the list does show, so the panel names both.
func TestWorkspaceContextFromComposeProject_CountsWhatTheListShows(t *testing.T) {
	p := docker.ComposeProject{Name: "web", Services: 3, Containers: 4, Running: 3, Exited: 1}

	withExtra := workspaceContextFromComposeProject(p, 4)
	if got := withExtra.lines[0]; got != "services: 3 of 4 in the file" {
		t.Errorf("expected the defined total named, got %q", got)
	}

	// Nothing extra to say when they agree.
	same := workspaceContextFromComposeProject(p, 3)
	if got := same.lines[0]; got != "services: 3" {
		t.Errorf("expected the plain count, got %q", got)
	}

	// And nothing when the row count is lower or missing, which the panel
	// reaches with the projects list unloaded: the two numbers come from
	// different sources there, and "3 of 0 defined" is worse than silence.
	for _, lower := range []int{0, 1} {
		if got := workspaceContextFromComposeProject(p, lower).lines[0]; got != "services: 3" {
			t.Errorf("with a row count of %d, expected the plain count, got %q", lower, got)
		}
	}
}

// The panel is reachable on a service the compose file defines and nothing
// runs, where every count is zero and the health summary reads "empty". On
// its own that reads as a broken service rather than one never started.
func TestWorkspaceContextFromComposeService_NamesANotCreatedService(t *testing.T) {
	notCreated := workspaceContextFromComposeService(docker.ComposeService{Project: "web", Name: "cache"})
	found := false
	for _, l := range notCreated.lines {
		if l == "state: absent, u brings it up" {
			found = true
		}
		if strings.HasPrefix(l, "status ratio:") {
			t.Errorf("expected no ratio for a service with no containers, got %q", l)
		}
	}
	if !found {
		t.Errorf("expected the absent state named, got %q", notCreated.lines)
	}
	// And nothing that describes containers there are none of: "health
	// summary: empty" under "not created" is a second, worse diagnosis.
	for _, l := range notCreated.lines {
		if strings.HasPrefix(l, "health summary:") {
			t.Errorf("expected no health summary for a service with no containers, got %q", l)
		}
	}

	// And a service that has containers keeps the ratio and says nothing
	// about being created, which it plainly is.
	running := workspaceContextFromComposeService(docker.ComposeService{
		Project: "web", Name: "api", Containers: 2, Running: 2,
	})
	ratio := false
	for _, l := range running.lines {
		if l == "status ratio: 2/2 running" {
			ratio = true
		}
		if strings.Contains(l, "absent") {
			t.Errorf("expected nothing about absence for a running service, got %q", l)
		}
	}
	if !ratio {
		t.Errorf("expected the status ratio, got %q", running.lines)
	}
	summary := false
	for _, l := range running.lines {
		if l == "health summary: healthy" {
			summary = true
		}
	}
	if !summary {
		t.Errorf("expected the health summary kept for a running service, got %q", running.lines)
	}
}

// The panel sits beside a list, so its count has to come from the model
// drawing that list. Both compose models keep their own drift map, and only
// the services model knows its first load has not landed, so reading the
// projects model in the Services view printed "3 of 4 defined" beside a
// list that still said it was loading.
func TestWorkspacePanel_DefinedCountComesFromTheVisibleList(t *testing.T) {
	appui.InitStyles()
	project := docker.ComposeProject{Name: "web", Services: 3, Containers: 3, Running: 3}
	projects := []docker.ProjectWithServices{{
		Project: project,
		Services: []docker.ComposeService{
			{Project: "web", Name: "api", Containers: 1, Running: 1},
			{Project: "web", Name: "db", Containers: 1, Running: 1},
			{Project: "web", Name: "worker", Containers: 1, Running: 1},
		},
	}}
	drift := map[string]map[string]docker.ServiceSync{"web": {"cache": docker.ServiceNotCreated}}

	// In the projects list, the projects model is the one drawing rows.
	m := newTestModel()
	m.view = ComposeProjects
	m.composeProjects.SetSize(120, 40)
	m.composeProjects.SetProjects(projects)
	m.composeProjects.SetDrift(drift)
	if got := m.definedServiceCount("web"); got != 4 {
		t.Errorf("expected the projects list's four rows, got %d", got)
	}

	// Entering the project, the services model draws the rows and has not
	// loaded yet, so there is no row count to report.
	m.view = ComposeServices
	m.selectedProject = "web"
	m.composeServices.SetSize(120, 40)
	if !m.composeServices.Loading() {
		t.Fatal("precondition: expected a fresh services model to be loading")
	}
	if got := m.definedServiceCount("web"); got != 0 {
		t.Errorf("expected no count while the services view loads, got %d", got)
	}
	if line := workspaceContextFromComposeProject(project, m.definedServiceCount("web")).lines[0]; line != "services: 3" {
		t.Errorf("expected the plain count beside a loading list, got %q", line)
	}

	// Once its own load lands, the count is that view's rows.
	m.composeServices.SetServices(projects[0].Services, nil, nil, "web")
	m.composeServices.SetDrift(drift)
	if got := m.definedServiceCount("web"); got != 4 {
		t.Errorf("expected the services view's four rows, got %d", got)
	}
}
