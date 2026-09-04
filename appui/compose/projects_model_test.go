package compose

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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

// A service the compose file defines and no container exists for had no row
// at all before this, because every row in this view comes from a
// container.
func TestProjectsModel_AServiceInTheFileWithNoContainersGetsARow(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(140, 20)
	m.SetProjects([]docker.ProjectWithServices{{
		Project:  docker.ComposeProject{Name: "web", Containers: 1, Running: 1},
		Services: []docker.ComposeService{{Project: "web", Name: "api", Containers: 1, Running: 1}},
	}})

	m.SetDrift(map[string]map[string]docker.ServiceSync{"web": {
		"api":    docker.ServiceInSync,
		"db":     docker.ServiceNotCreated,
		"worker": docker.ServiceNotCreated,
	}})

	var names []string
	for _, row := range m.table.FilteredRows() {
		names = append(names, strings.TrimSpace(ansi.Strip(row.Columns()[0])))
	}
	// The running service first, then the file's own.
	want := []string{"web", "api", "db", "worker"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("expected rows %v, got %v", want, names)
	}
	// And the row says what it is.
	if view := ansi.Strip(m.View()); !strings.Contains(view, "absent") {
		t.Errorf("expected the not-created rows to report SYNC absent:\n%s", view)
	}
}

// A running service must not also appear as not created, whatever the drift
// map says about it.
func TestProjectsModel_ARunningServiceIsNotDuplicated(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(140, 20)
	m.SetProjects([]docker.ProjectWithServices{{
		Project:  docker.ComposeProject{Name: "web"},
		Services: []docker.ComposeService{{Project: "web", Name: "api", Containers: 1}},
	}})
	m.SetDrift(map[string]map[string]docker.ServiceSync{"web": {"api": docker.ServiceNotCreated}})

	count := 0
	for _, row := range m.table.FilteredRows() {
		if strings.TrimSpace(ansi.Strip(row.Columns()[0])) == "api" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected one row for api, got %d", count)
	}
}

// The three reasons a view can be empty are indistinguishable otherwise.
func TestProjectsModel_AnEmptyViewSaysWhy(t *testing.T) {
	appui.InitStyles()
	body := func(m ProjectsModel) string {
		return strings.TrimRight(ansi.Strip(strings.Split(m.View(), "\n")[3]), " ")
	}

	m := NewProjectsModel()
	// 58 columns, not 140: that is what workspace mode leaves the list on a
	// 100-column terminal, and the half that says where dry looks is the
	// half a longer message loses.
	m.SetSize(58, 8)
	if got := body(m); !strings.Contains(got, "Loading") {
		t.Errorf("before any load, expected a loading message, got %q", got)
	}

	m.SetProjects(nil)
	got := body(m)
	if strings.Contains(got, "Loading") || !strings.Contains(got, "No compose projects") {
		t.Errorf("loaded and empty, expected to say so, got %q", got)
	}
	// The second half is the useful half, and it has to survive at 80
	// columns: truncated away, the message says only that there is nothing.
	if !strings.Contains(got, "labels") || !strings.Contains(got, "directory") {
		t.Errorf("expected the message to say where dry looks, got %q", got)
	}
	if strings.HasSuffix(got, "…") {
		t.Errorf("expected the message to fit 58 columns, got %q", got)
	}

	// Through the keys, not the SetFilter shim: the message is pushed from
	// Update's filter branch, and a test that calls SetFilter directly
	// passes with that push deleted.
	m.SetProjects([]docker.ProjectWithServices{{Project: docker.ComposeProject{Name: "web"}}})
	m, _ = m.Update(tea.KeyPressMsg{Code: '%'})
	for _, r := range "nothing-matches" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if got := body(m); !strings.Contains(got, "filter") {
		t.Errorf("filtered to nothing, expected the filter named, got %q", got)
	}
}

// A drift result can insert a row above the selection, which shifts every
// row below it; see refreshRows for what a slid cursor costs.
func TestProjectsModel_ANewRowAboveTheCursorDoesNotMoveTheSelection(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(140, 20)
	m.SetProjects([]docker.ProjectWithServices{
		{
			Project:  docker.ComposeProject{Name: "aaa"},
			Services: []docker.ComposeService{{Project: "aaa", Name: "api", Containers: 1}},
		},
		{
			Project:  docker.ComposeProject{Name: "zzz"},
			Services: []docker.ComposeService{{Project: "zzz", Name: "web", Containers: 1}},
		},
	})
	// Walk to the last row: zzz's own service.
	for range 3 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	svc := m.SelectedService()
	if svc == nil || svc.Name != "web" {
		t.Fatalf("precondition: expected zzz/web selected, got %+v", svc)
	}

	// A cache service appears in aaa, above the selection.
	m.SetDrift(map[string]map[string]docker.ServiceSync{
		"aaa": {"api": docker.ServiceInSync, "cache": docker.ServiceNotCreated},
	})

	after := m.SelectedService()
	if after == nil {
		t.Fatal("the selection was lost, so the cursor now points at a project header")
	}
	if after.Name != "web" || after.Project != "zzz" {
		t.Errorf("expected zzz/web still selected, got %s/%s", after.Project, after.Name)
	}
}

// The order the rows come back in has to be stable, or a refresh shuffles
// them under the cursor. Asserted on the function rather than through a
// view, and repeated, because Go randomises map iteration: a missing sort
// can return the right order by chance, so one pass proves nothing. On
// go1.27 these six names come back in six distinct orders and never the
// sorted one, but that is the runtime's business and not a promise, so the
// repetition carries the check rather than the choice of names. Correct
// code passes every iteration, so the loop cannot make this flaky.
func TestNotCreatedServices_ComeBackInAStableOrder(t *testing.T) {
	sync := map[string]docker.ServiceSync{}
	for _, name := range []string{"foxtrot", "bravo", "delta", "alpha", "echo", "charlie"} {
		sync[name] = docker.ServiceNotCreated
	}
	want := "alpha,bravo,charlie,delta,echo,foxtrot"

	for i := range 64 {
		var got []string
		for _, svc := range notCreatedServices("web", nil, sync) {
			got = append(got, svc.Name)
		}
		if strings.Join(got, ",") != want {
			t.Fatalf("pass %d: expected %s, got %v", i, want, got)
		}
	}
}

// The row the cursor was on can disappear: a service renamed in the compose
// file, or a project whose config momentarily fails to resolve. The cursor
// then has to stay inside the project that row belonged to. u on a service
// row brings that service up with no confirmation, so a cursor that slides
// into another project's service brings up something the user never
// selected. The shapes below are the ones a forward-only search gets
// wrong: the lost row was the last, or every row below it is another
// project's header.
func TestProjectsModel_ALostRowKeepsTheCursorInItsOwnProject(t *testing.T) {
	appui.InitStyles()
	project := func(name string, services ...string) docker.ProjectWithServices {
		p := docker.ProjectWithServices{Project: docker.ComposeProject{Name: name}}
		for _, s := range services {
			p.Services = append(p.Services, docker.ComposeService{Project: name, Name: s, Containers: 1})
		}
		return p
	}

	cases := []struct {
		name     string
		projects []docker.ProjectWithServices
		lost     string // the project whose cache row goes away
		down     int    // keypresses to reach that row
		want     string // project the cursor must still be in
		wantSvc  string // service the cursor must land on, when one is the right answer
	}{
		{
			name:     "the lost row was the last row",
			projects: []docker.ProjectWithServices{project("aaa", "api"), project("zzz")},
			lost:     "zzz", down: 3, want: "zzz",
		},
		{
			name:     "every row below is another project's header",
			projects: []docker.ProjectWithServices{project("aaa"), project("mmm"), project("zzz")},
			lost:     "aaa", down: 1, want: "aaa",
		},
		{
			name:     "a service row exists but in a later project",
			projects: []docker.ProjectWithServices{project("aaa"), project("mmm"), project("zzz", "web")},
			lost:     "aaa", down: 1, want: "aaa",
		},
		{
			name:     "the project has another service to fall back to",
			projects: []docker.ProjectWithServices{project("aaa", "api"), project("zzz", "web")},
			lost:     "aaa", down: 2, want: "aaa", wantSvc: "api",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewProjectsModel()
			m.SetSize(140, 20)
			m.SetProjects(tc.projects)
			m.SetDrift(map[string]map[string]docker.ServiceSync{
				tc.lost: {"cache": docker.ServiceNotCreated},
			})
			for range tc.down {
				m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			}
			svc := m.SelectedService()
			if svc == nil || svc.Name != "cache" || svc.Project != tc.lost {
				t.Fatalf("precondition: expected %s/cache selected, got %+v", tc.lost, svc)
			}

			// The compose file no longer defines it.
			m.SetDrift(map[string]map[string]docker.ServiceSync{})

			got := m.SelectedProject()
			if got == nil {
				t.Fatal("the selection was lost entirely")
			}
			if got.Name != tc.want {
				t.Errorf("cursor moved to project %s, expected to stay in %s: u acts there",
					got.Name, tc.want)
			}
			if tc.wantSvc == "" {
				return
			}
			svc = m.SelectedService()
			if svc == nil || svc.Name != tc.wantSvc {
				t.Errorf("expected the project's remaining service %s under the cursor, got %+v",
					tc.wantSvc, svc)
			}
		})
	}
}

// When the followed row is gone but the index now holds another service,
// the cursor stays there: the step-off exists for a header, and moving off a
// perfectly good service row would be the slide it prevents.
func TestProjectsModel_ALostRowKeepsAServiceUnderTheCursor(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(140, 20)
	m.SetProjects([]docker.ProjectWithServices{
		{
			Project:  docker.ComposeProject{Name: "aaa"},
			Services: []docker.ComposeService{{Project: "aaa", Name: "api", Containers: 1}},
		},
		{
			Project:  docker.ComposeProject{Name: "zzz"},
			Services: []docker.ComposeService{{Project: "zzz", Name: "web", Containers: 1}},
		},
	})
	m.SetDrift(map[string]map[string]docker.ServiceSync{"aaa": {
		"api":   docker.ServiceInSync,
		"cache": docker.ServiceNotCreated,
	}})
	// Cursor on aaa/api, with aaa/cache directly below it.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if svc := m.SelectedService(); svc == nil || svc.Name != "api" {
		t.Fatalf("precondition: expected aaa/api selected, got %+v", svc)
	}

	// api's containers go away and the file does not define it: its row is
	// gone, and aaa/cache slides into that index.
	m.SetProjects([]docker.ProjectWithServices{
		{Project: docker.ComposeProject{Name: "aaa"}},
		{
			Project:  docker.ComposeProject{Name: "zzz"},
			Services: []docker.ComposeService{{Project: "zzz", Name: "web", Containers: 1}},
		},
	})
	m.SetDrift(map[string]map[string]docker.ServiceSync{"aaa": {"cache": docker.ServiceNotCreated}})

	svc := m.SelectedService()
	if svc == nil {
		t.Fatal("expected a service still under the cursor")
	}
	if svc.Name != "cache" || svc.Project != "aaa" {
		t.Errorf("expected aaa's remaining service, aaa/cache, got %s/%s", svc.Project, svc.Name)
	}
}

// When the project is gone too there is no right answer, so the cursor
// settles on any row a key can act on per service. Searching only downwards
// leaves it on a project header whenever the rows below are all headers,
// which is what the projects with nothing running look like.
func TestProjectsModel_ALostProjectSettlesOnAServiceRow(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(140, 20)
	m.SetProjects([]docker.ProjectWithServices{
		{
			Project:  docker.ComposeProject{Name: "aaa"},
			Services: []docker.ComposeService{{Project: "aaa", Name: "api", Containers: 1}},
		},
		{Project: docker.ComposeProject{Name: "mmm"}},
		{Project: docker.ComposeProject{Name: "zzz"}},
	})
	m.SetDrift(map[string]map[string]docker.ServiceSync{"mmm": {"cache": docker.ServiceNotCreated}})
	// Rows: aaa, aaa/api, mmm, mmm/cache, zzz. Land on mmm/cache, whose only
	// neighbour below is zzz's header.
	for range 3 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if svc := m.SelectedService(); svc == nil || svc.Project != "mmm" {
		t.Fatalf("precondition: expected mmm/cache selected, got %+v", svc)
	}

	// mmm disappears entirely.
	m.SetProjects([]docker.ProjectWithServices{
		{
			Project:  docker.ComposeProject{Name: "aaa"},
			Services: []docker.ComposeService{{Project: "aaa", Name: "api", Containers: 1}},
		},
		{Project: docker.ComposeProject{Name: "zzz"}},
	})
	m.SetDrift(map[string]map[string]docker.ServiceSync{})

	svc := m.SelectedService()
	if svc == nil {
		p := m.SelectedProject()
		name := "<none>"
		if p != nil {
			name = p.Name
		}
		t.Fatalf("cursor left on the %s project header, where per-service keys cannot act", name)
	}
	if svc.Name != "api" {
		t.Errorf("expected the nearest service row, aaa/api, got %s/%s", svc.Project, svc.Name)
	}
}

// Only the not-created label may conjure a row. The other verdicts describe
// a service that has containers, so a row for one would be a duplicate of a
// row already there, or a claim about a service nothing knows about.
func TestNotCreatedServices_OnlyTheNotCreatedLabelMakesARow(t *testing.T) {
	for _, status := range []docker.ServiceSync{
		docker.ServiceInSync, docker.ServiceDrifted, docker.ServiceUnknown, docker.ServiceSync(""),
	} {
		got := notCreatedServices("web", nil, map[string]docker.ServiceSync{"cache": status})
		if len(got) != 0 {
			t.Errorf("status %q produced a row: %+v", status, got)
		}
	}
	if got := notCreatedServices("web", nil, map[string]docker.ServiceSync{"cache": docker.ServiceNotCreated}); len(got) != 1 {
		t.Errorf("expected the not-created service to produce a row, got %+v", got)
	}
}

// The filter, escaping the filter, and f1 all change the visible row set,
// so each has to follow the row too. Before this, escaping a filter left
// the cursor on whatever index it held, which is usually a project header.
func TestProjectsModel_FilterAndSortFollowTheSelectedRow(t *testing.T) {
	appui.InitStyles()
	newModel := func() ProjectsModel {
		m := NewProjectsModel()
		m.SetSize(140, 20)
		m.SetProjects([]docker.ProjectWithServices{
			{
				Project:  docker.ComposeProject{Name: "alpha"},
				Services: []docker.ComposeService{{Project: "alpha", Name: "a1", Containers: 1}},
			},
			{
				Project:  docker.ComposeProject{Name: "beta"},
				Services: []docker.ComposeService{{Project: "beta", Name: "b1", Containers: 1}},
			},
		})
		return m
	}
	selected := func(m ProjectsModel) string {
		if svc := m.SelectedService(); svc != nil {
			return svc.Project + "/" + svc.Name
		}
		if p := m.SelectedProject(); p != nil {
			return "project " + p.Name
		}
		return "<none>"
	}

	t.Run("escaping a filter", func(t *testing.T) {
		m := newModel()
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		want := selected(m)
		if want != "beta/b1" {
			t.Fatalf("precondition: expected beta/b1, got %s", want)
		}
		m, _ = m.Update(tea.KeyPressMsg{Code: '%'})
		for _, r := range "b1" {
			m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		}
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		if got := selected(m); got != want {
			t.Errorf("after escaping the filter, selection moved from %s to %s", want, got)
		}
	})

	t.Run("sorting", func(t *testing.T) {
		m := newModel()
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		want := selected(m)
		if want != "alpha/a1" {
			t.Fatalf("precondition: expected alpha/a1, got %s", want)
		}
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
		if got := selected(m); got != want {
			t.Errorf("after f1, selection moved from %s to %s", want, got)
		}
	})
}

// The same step-off, one row earlier: when the cursor is on the first row
// there is nothing above it, so a backward-only walk leaves it on a project
// header, where every per-service key has nothing to act on.
func TestProjectsModel_ALostFirstProjectStepsForwardToAService(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(140, 20)
	m.SetProjects([]docker.ProjectWithServices{
		{Project: docker.ComposeProject{Name: "aaa"}},
		{
			Project:  docker.ComposeProject{Name: "zzz"},
			Services: []docker.ComposeService{{Project: "zzz", Name: "web", Containers: 1}},
		},
	})
	// Rows: aaa, zzz, zzz/web. Put the cursor on aaa's header, the first row.
	m.SetFilter("aaa")
	m.SetFilter("")
	if p := m.SelectedProject(); p == nil || p.Name != "aaa" {
		t.Fatalf("precondition: expected aaa's header selected, got %+v", p)
	}

	// aaa disappears, so the cursor's index now holds zzz's header and there
	// is no row above it to fall back to.
	m.SetProjects([]docker.ProjectWithServices{
		{
			Project:  docker.ComposeProject{Name: "zzz"},
			Services: []docker.ComposeService{{Project: "zzz", Name: "web", Containers: 1}},
		},
	})

	svc := m.SelectedService()
	if svc == nil {
		p := m.SelectedProject()
		name := "<none>"
		if p != nil {
			name = p.Name
		}
		t.Fatalf("cursor left on the %s project header, where per-service keys cannot act", name)
	}
	if svc.Name != "web" {
		t.Errorf("expected the only service row, zzz/web, got %s/%s", svc.Project, svc.Name)
	}
}

// The step-off runs only when the cursor is on a header. Reaching it means
// the followed row and its whole project are gone, and the row the clamp
// left under the cursor can be a perfectly good service of a third project.
// Moving off that row would be the slide the step-off exists to prevent.
func TestProjectsModel_ALostProjectLeavesAGoodServiceRowAlone(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(140, 20)
	m.SetProjects([]docker.ProjectWithServices{
		{
			Project: docker.ComposeProject{Name: "aaa"},
			Services: []docker.ComposeService{
				{Project: "aaa", Name: "api", Containers: 1},
				{Project: "aaa", Name: "db", Containers: 1},
			},
		},
		{
			Project:  docker.ComposeProject{Name: "zzz"},
			Services: []docker.ComposeService{{Project: "zzz", Name: "web", Containers: 1}},
		},
	})
	// Rows: aaa, aaa/api, aaa/db, zzz, zzz/web. Land on the last one.
	for range 4 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if svc := m.SelectedService(); svc == nil || svc.Project != "zzz" {
		t.Fatalf("precondition: expected zzz/web selected, got %+v", svc)
	}

	// zzz goes away entirely, so the cursor clamps onto aaa/db, the last
	// row of the shorter list and a service row in its own right.
	m.SetProjects([]docker.ProjectWithServices{{
		Project: docker.ComposeProject{Name: "aaa"},
		Services: []docker.ComposeService{
			{Project: "aaa", Name: "api", Containers: 1},
			{Project: "aaa", Name: "db", Containers: 1},
		},
	}})

	svc := m.SelectedService()
	if svc == nil {
		t.Fatal("expected a service under the cursor")
	}
	if svc.Name != "db" {
		t.Errorf("expected the row the clamp left, aaa/db, got %s/%s", svc.Project, svc.Name)
	}
}

// Project and service names both come from container labels the user sets,
// so the two row kinds have to be told apart by something other than the
// name. Unprefixed, project "aaa/api" and project aaa's service api are the
// same ID, and following the selection picks whichever comes first.
func TestProjectsModel_ARowIDCannotCollideAcrossKinds(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(160, 20)
	m.SetProjects([]docker.ProjectWithServices{
		// A project whose name is another project's service row.
		{
			Project:  docker.ComposeProject{Name: "aaa"},
			Services: []docker.ComposeService{{Project: "aaa", Name: "api", Containers: 1}},
		},
		{Project: docker.ComposeProject{Name: "aaa/api"}},
		// And two service rows whose project and service names split the
		// same text at different points.
		{
			Project:  docker.ComposeProject{Name: "b"},
			Services: []docker.ComposeService{{Project: "b", Name: "c/d", Containers: 1}},
		},
		{
			Project:  docker.ComposeProject{Name: "b/c"},
			Services: []docker.ComposeService{{Project: "b/c", Name: "d", Containers: 1}},
		},
	})

	seen := map[string]int{}
	for _, row := range m.table.FilteredRows() {
		seen[row.ID()]++
	}
	for id, n := range seen {
		if n > 1 {
			t.Errorf("row ID %q appears %d times", id, n)
		}
	}
	// Four headers and three services.
	if len(seen) != 7 {
		t.Errorf("expected seven distinct row IDs, got %d: %v", len(seen), seen)
	}

	// The NUL is what makes the IDs injective; the kind prefix is what
	// makes them readable in a failure message, and it is asserted so that
	// dropping it is a test failure rather than a silent style change.
	for _, row := range m.table.FilteredRows() {
		switch row.(type) {
		case projectHeaderRow:
			if !strings.HasPrefix(row.ID(), "project:") {
				t.Errorf("header ID %q is not prefixed by its kind", row.ID())
			}
		case serviceDetailRow:
			if !strings.HasPrefix(row.ID(), "service:") {
				t.Errorf("service ID %q is not prefixed by its kind", row.ID())
			}
			if !strings.Contains(row.ID(), "\x00") {
				t.Errorf("service ID %q joins its halves with something a name can contain", row.ID())
			}
		}
	}
}

// A drift map with a nameless key would put a row on screen with no name,
// selectable and actionable like any other.
func TestNotCreatedServices_ANamelessServiceMakesNoRow(t *testing.T) {
	got := notCreatedServices("web", nil, map[string]docker.ServiceSync{
		"":      docker.ServiceNotCreated,
		"cache": docker.ServiceNotCreated,
	})
	if len(got) != 1 || got[0].Name != "cache" {
		t.Errorf("expected cache alone, got %+v", got)
	}
}

// f1 cycles the sort field. The rows are grouped, so sorting them flat, as
// the table does for every ungrouped view, moves services under other
// projects' headers and leaves service rows above the first header. Every
// field is tried, because the corruption showed on the second field, not
// the default one.
func TestProjectsModel_SortKeepsEveryServiceUnderItsOwnProject(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(160, 20)
	m.SetProjects([]docker.ProjectWithServices{
		{
			Project: docker.ComposeProject{Name: "zzz", Containers: 3},
			Services: []docker.ComposeService{
				{Project: "zzz", Name: "api", Containers: 3, Running: 3},
				{Project: "zzz", Name: "web", Containers: 1},
			},
		},
		{
			Project:  docker.ComposeProject{Name: "aaa", Containers: 1},
			Services: []docker.ComposeService{{Project: "aaa", Name: "db", Containers: 1}},
		},
	})
	m.SetDrift(map[string]map[string]docker.ServiceSync{"aaa": {"cache": docker.ServiceNotCreated}})

	// Sixteen presses: both tables have eight columns, so this is every
	// field twice round, and a field that only misbehaves on a second
	// rebuild is caught too.
	for press := range 16 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
		rows := m.table.FilteredRows()
		if _, first := rows[0].(projectHeaderRow); !first {
			t.Fatalf("press %d: the first row is not a project header: %s", press, rows[0].ID())
		}
		project := ""
		counted := map[string]int{}
		for _, row := range rows {
			switch r := row.(type) {
			case projectHeaderRow:
				project = r.project.Name
			case serviceDetailRow:
				if r.projectName != project {
					t.Fatalf("press %d: service %s/%s sits under project %s",
						press, r.projectName, r.service.Name, project)
				}
				counted[project]++
			}
		}
		if counted["zzz"] != 2 || counted["aaa"] != 2 {
			t.Fatalf("press %d: expected two services under each project, got %v", press, counted)
		}
	}

	// And the sort actually reaches the services: on CONTAINERS ascending,
	// zzz's rows go web (1) then api (3), the reverse of their name order.
	m.table.SetSortField(0)
	m.refreshRows()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	var order []string
	for _, row := range m.table.FilteredRows() {
		if r, ok := row.(serviceDetailRow); ok && r.projectName == "zzz" {
			order = append(order, r.service.Name)
		}
	}
	if strings.Join(order, ",") != "web,api" {
		t.Errorf("expected zzz sorted by container count, web then api, got %v", order)
	}
}

// The old flat sort was thrown away by the next SetRows, so f1 in this view
// held only until the next drift result landed. Ordering at build time
// means a refresh rebuilds in the order the user asked for.
func TestProjectsModel_SortSurvivesARefresh(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(160, 20)
	projects := []docker.ProjectWithServices{
		{
			Project: docker.ComposeProject{Name: "zzz", Containers: 3},
			Services: []docker.ComposeService{
				{Project: "zzz", Name: "api", Containers: 3},
				{Project: "zzz", Name: "web", Containers: 1},
			},
		},
		{
			Project:  docker.ComposeProject{Name: "aaa", Containers: 1},
			Services: []docker.ComposeService{{Project: "aaa", Name: "db", Containers: 1}},
		},
	}
	m.SetProjects(projects)

	// Sort by CONTAINERS: aaa (1) before zzz (3), and web before api.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	order := func() string {
		var out []string
		for _, row := range m.table.FilteredRows() {
			switch r := row.(type) {
			case projectHeaderRow:
				out = append(out, "["+r.project.Name+"]")
			case serviceDetailRow:
				out = append(out, r.service.Name)
			}
		}
		return strings.Join(out, " ")
	}
	want := "[aaa] db [zzz] web api"
	if got := order(); got != want {
		t.Fatalf("after f1 expected %q, got %q", want, got)
	}

	// A drift result arrives, which rebuilds every row.
	m.SetDrift(map[string]map[string]docker.ServiceSync{"zzz": {"api": docker.ServiceInSync}})
	if got := order(); got != want {
		t.Errorf("after a refresh expected the sort to hold, %q, got %q", want, got)
	}

	// So does a new project list.
	m.SetProjects(projects)
	if got := order(); got != want {
		t.Errorf("after a reload expected the sort to hold, %q, got %q", want, got)
	}
}

// The merge by name is the tiebreak once the sort is on some other column.
// Sorting on a column where every service cell is empty leaves the order
// the rows were built in, so building the file-only ones on the end rather
// than in among the rest is visible there.
func TestProjectsModel_ServicesTieBreakByName(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(160, 20)
	m.SetProjects([]docker.ProjectWithServices{{
		Project:  docker.ComposeProject{Name: "web"},
		Services: []docker.ComposeService{{Project: "web", Name: "zzz", Containers: 1}},
	}})
	m.SetDrift(map[string]map[string]docker.ServiceSync{"web": {"aaa": docker.ServiceNotCreated}})

	// PORTS, the last column, is empty on both rows.
	m.table.SetSortField(7)
	m.refreshRows()

	var order []string
	for _, row := range m.table.FilteredRows() {
		if r, ok := row.(serviceDetailRow); ok {
			order = append(order, r.service.Name)
		}
	}
	if strings.Join(order, ",") != "aaa,zzz" {
		t.Errorf("expected the name order to break the tie, aaa then zzz, got %v", order)
	}
}

// The fallback prefers the nearest remaining service of the project, not
// the first one: with the cursor at the bottom of a long project, the first
// is a screen away from where the user was looking.
func TestProjectsModel_ALostRowFallsBackToTheNearestService(t *testing.T) {
	appui.InitStyles()
	services := func(names ...string) []docker.ComposeService {
		var out []docker.ComposeService
		for _, n := range names {
			out = append(out, docker.ComposeService{Project: "web", Name: n, Containers: 1})
		}
		return out
	}
	m := NewProjectsModel()
	m.SetSize(160, 20)
	m.SetProjects([]docker.ProjectWithServices{{
		Project:  docker.ComposeProject{Name: "web"},
		Services: services("a1", "a2", "a3", "a4"),
	}})
	m.SetDrift(map[string]map[string]docker.ServiceSync{"web": {"a5": docker.ServiceNotCreated}})
	// Rows: web, a1, a2, a3, a4, a5. Land on a5, the last one.
	for range 5 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if svc := m.SelectedService(); svc == nil || svc.Name != "a5" {
		t.Fatalf("precondition: expected web/a5 selected, got %+v", svc)
	}

	// The file stops defining a5, so its row goes.
	m.SetDrift(map[string]map[string]docker.ServiceSync{})

	svc := m.SelectedService()
	if svc == nil {
		t.Fatal("expected a service still under the cursor")
	}
	if svc.Name != "a4" {
		t.Errorf("expected the nearest remaining service a4, got %s", svc.Name)
	}
}

// A project header is a row the cursor can rest on deliberately, so it is
// followed across a refresh like any other. Reading its ID off the row is
// what makes that work: without it the refresh has nothing to follow, and
// the cursor keeps an index that now belongs to another project's service.
func TestProjectsModel_AProjectHeaderIsFollowedToo(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(160, 20)
	m.SetProjects([]docker.ProjectWithServices{
		{
			Project:  docker.ComposeProject{Name: "aaa"},
			Services: []docker.ComposeService{{Project: "aaa", Name: "api", Containers: 1}},
		},
		{
			Project:  docker.ComposeProject{Name: "zzz"},
			Services: []docker.ComposeService{{Project: "zzz", Name: "web", Containers: 1}},
		},
	})
	// Rows: aaa, aaa/api, zzz, zzz/web. Land on zzz's header.
	for range 2 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	p := m.SelectedProject()
	if p == nil || p.Name != "zzz" {
		t.Fatalf("precondition: expected zzz's header selected, got %+v", p)
	}
	if m.SelectedService() != nil {
		t.Fatal("precondition: expected a header, not a service")
	}

	// A drift result adds a row to aaa, above the cursor.
	m.SetDrift(map[string]map[string]docker.ServiceSync{"aaa": {"cache": docker.ServiceNotCreated}})

	after := m.SelectedProject()
	if after == nil || after.Name != "zzz" {
		t.Errorf("expected zzz's header still selected, got %+v", after)
	}
	if svc := m.SelectedService(); svc != nil {
		t.Errorf("expected the header still selected, not the service %s/%s", svc.Project, svc.Name)
	}
}

// The list is built in the order of the active sort field, which on load is
// the default one, NAME. AggregateComposeProjects already returns projects
// and services in that order, but the working-directory scan appends to
// what the labels found, so what arrives is not always sorted.
func TestProjectsModel_LoadsInNameOrder(t *testing.T) {
	appui.InitStyles()
	m := NewProjectsModel()
	m.SetSize(160, 20)
	m.SetProjects([]docker.ProjectWithServices{
		{
			Project: docker.ComposeProject{Name: "zzz"},
			Services: []docker.ComposeService{
				{Project: "zzz", Name: "web", Containers: 1},
				{Project: "zzz", Name: "api", Containers: 1},
			},
		},
		{
			Project:  docker.ComposeProject{Name: "aaa"},
			Services: []docker.ComposeService{{Project: "aaa", Name: "db", Containers: 1}},
		},
	})

	var order []string
	for _, row := range m.table.FilteredRows() {
		switch r := row.(type) {
		case projectHeaderRow:
			order = append(order, "["+r.project.Name+"]")
		case serviceDetailRow:
			order = append(order, r.service.Name)
		}
	}
	if want := "[aaa] db [zzz] api web"; strings.Join(order, " ") != want {
		t.Errorf("expected %q on load, got %q", want, strings.Join(order, " "))
	}
}
