package compose

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

// TestServicesModel_SetDrift_RendersSyncColumn is the services-view sibling
// of the projects-view drift rendering test: a duplicate colorSync/SYNC
// column exists in this file's newServiceRow, so a bug in only this model
// would go uncaught if only the projects model were tested.
func TestServicesModel_SetDrift_RendersSyncColumn(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices([]docker.ComposeService{
		{Project: "web", Name: "api"},
		{Project: "web", Name: "worker"},
	}, nil, nil, "web")
	m.SetDrift(map[string]map[string]docker.ServiceSync{
		"web": {
			"api":    docker.ServiceNotCreated,
			"worker": docker.ServiceInSync,
		},
	})

	var apiCols, workerCols []string
	for _, row := range m.table.FilteredRows() {
		r, ok := row.(serviceRow)
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

	const syncCol = 6 // NAME, CONTAINERS, RUNNING, EXITED, IMAGE/DRIVER, HEALTH/SCOPE, SYNC, PORTS
	if !strings.Contains(apiCols[syncCol], "none") {
		t.Fatalf("expected the not-created service's SYNC cell to render \"none\", got %q", apiCols[syncCol])
	}
	if !strings.Contains(workerCols[syncCol], "ok") {
		t.Fatalf("expected the in-sync service's SYNC cell to render \"ok\", got %q", workerCols[syncCol])
	}
}

// TestServicesModel_SetServices_NetworkAndVolumeRowsHaveEmptySyncCell guards
// finding-adjacent column alignment: network and volume rows must carry an
// empty SYNC cell in the same position as the header, or their remaining
// columns (PORTS in particular) shift left relative to the header.
func TestServicesModel_SetServices_NetworkAndVolumeRowsHaveEmptySyncCell(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices(nil,
		[]docker.ComposeNetwork{{Name: "web_default", Driver: "bridge", Scope: "local"}},
		[]docker.ComposeVolume{{Name: "web_data", Driver: "local"}},
		"web")

	var netCols, volCols []string
	for _, row := range m.table.FilteredRows() {
		switch r := row.(type) {
		case networkRow:
			netCols = r.Columns()
		case volumeRow:
			volCols = r.Columns()
		}
	}
	if netCols == nil || volCols == nil {
		t.Fatalf("expected both a network and a volume row, got net=%v vol=%v", netCols, volCols)
	}
	const syncCol = 6
	if netCols[syncCol] != "" {
		t.Fatalf("expected the network row's SYNC cell to be empty, got %q", netCols[syncCol])
	}
	if volCols[syncCol] != "" {
		t.Fatalf("expected the volume row's SYNC cell to be empty, got %q", volCols[syncCol])
	}
	// Ports column (index 7) must still be the last column for both.
	if len(netCols) != 8 || len(volCols) != 8 {
		t.Fatalf("expected 8 columns (NAME..PORTS) on both rows, got net=%d vol=%d", len(netCols), len(volCols))
	}
}

// --- Section headers are never the selection -------------------------------
//
// The Compose Services view prepends a "Services (n)" header row to each
// group. No key can act on one: u, ctrl+s/t/r/e, l and enter all resolve the
// cursor to a service, a network or a volume first. A cursor parked on a
// header therefore makes every one of those keys do nothing, so the model
// keeps the cursor off headers, on a fresh load, and while navigating.

// fixture builds a view holding two services, one network and one volume,
// i.e. all three sections and their three headers.
func fixture(t *testing.T) ServicesModel {
	t.Helper()
	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices(
		[]docker.ComposeService{
			{Project: "web", Name: "api"},
			{Project: "web", Name: "worker"},
		},
		[]docker.ComposeNetwork{{Name: "web_default", Driver: "bridge", Scope: "local"}},
		[]docker.ComposeVolume{{Name: "web_data", Driver: "local"}},
		"web")
	return m
}

// highlightedLine returns the rendered line the inner table marks as
// selected, stripped of styling, or "" when nothing on screen is marked.
// A selection the view does not render is the bug it exists to catch.
func highlightedLine(m ServicesModel) string {
	marker := strings.SplitN(appui.SelectedRowStyle.Render("x"), "x", 2)[0]
	for _, line := range strings.Split(m.View(), "\n") {
		if strings.Contains(line, marker) {
			return ansi.Strip(line)
		}
	}
	return ""
}

// selectionOf names what the cursor currently resolves to, for assertions
// that care about the row's identity rather than its type.
func selectionOf(m ServicesModel) string {
	if s := m.SelectedService(); s != nil {
		return "service:" + s.Name
	}
	if n := m.SelectedNetwork(); n != nil {
		return "network:" + n.Name
	}
	if v := m.SelectedVolume(); v != nil {
		return "volume:" + v.Name
	}
	row := m.table.SelectedRow()
	if row == nil {
		return "none"
	}
	return "unselectable:" + row.ID()
}

func TestServicesModel_FreshLoadSelectsTheFirstService(t *testing.T) {
	m := fixture(t)
	if got := selectionOf(m); got != "service:api" {
		t.Fatalf("a fresh view must select the first service, got %s", got)
	}
}

// A project can have no services yet still have networks and volumes (its
// containers are gone but `down --volumes` was never run). The first
// selectable row is then a network.
func TestServicesModel_FreshLoadWithNoServicesSelectsTheFirstNetwork(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices(nil,
		[]docker.ComposeNetwork{{Name: "web_default", Driver: "bridge", Scope: "local"}},
		[]docker.ComposeVolume{{Name: "web_data", Driver: "local"}},
		"web")
	if got := selectionOf(m); got != "network:web_default" {
		t.Fatalf("expected the first network to be selected, got %s", got)
	}
}

// Walking down from the top must visit every resource row and never stop on
// a header, whatever the section boundaries look like.
func TestServicesModel_DownNeverStopsOnASectionHeader(t *testing.T) {
	m := fixture(t)
	want := []string{
		"service:api",
		"service:worker",
		"network:web_default",
		"volume:web_data",
	}
	var got []string
	got = append(got, selectionOf(m))
	for range m.table.FilteredRows() {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		if s := selectionOf(m); s != got[len(got)-1] {
			got = append(got, s)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("expected to visit %v, visited %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected to visit %v, visited %v", want, got)
		}
	}
}

// Coming back up over a section boundary must skip the header backwards, not
// stall on it: the "Networks" header sits between web_default and worker.
func TestServicesModel_UpSkipsBackOverASectionHeader(t *testing.T) {
	m := fixture(t)
	for range 2 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if got := selectionOf(m); got != "network:web_default" {
		t.Fatalf("precondition: expected the network to be selected, got %s", got)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if got := selectionOf(m); got != "service:worker" {
		t.Fatalf("expected up to skip the Networks header onto worker, got %s", got)
	}
}

// g/home jumps the inner table to row 0, which is always a header: the jump
// has to carry on forward to the first service.
func TestServicesModel_GotoTopLandsOnAService(t *testing.T) {
	m := fixture(t)
	for range 3 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if got := selectionOf(m); got != "volume:web_data" {
		t.Fatalf("precondition: expected the volume to be selected, got %s", got)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'g'})
	if got := selectionOf(m); got != "service:api" {
		t.Fatalf("expected g to land on the first service, got %s", got)
	}
}

// A reload while the user sits on a service must not yank the cursor back to
// the top: compose reloads on every container event, so a cursor that reset
// itself would be unusable during an up.
func TestServicesModel_ReloadKeepsTheCursorOnItsService(t *testing.T) {
	m := fixture(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := selectionOf(m); got != "service:worker" {
		t.Fatalf("precondition: expected worker to be selected, got %s", got)
	}
	m.SetServices(
		[]docker.ComposeService{
			{Project: "web", Name: "api"},
			{Project: "web", Name: "worker"},
		},
		[]docker.ComposeNetwork{{Name: "web_default", Driver: "bridge", Scope: "local"}},
		[]docker.ComposeVolume{{Name: "web_data", Driver: "local"}},
		"web")
	if got := selectionOf(m); got != "service:worker" {
		t.Fatalf("expected the cursor to stay on worker across a reload, got %s", got)
	}
}

// Drift arriving rebuilds every row. The cursor must survive that too, or
// the SYNC column landing would move the selection under the user's hands.
func TestServicesModel_DriftKeepsTheCursorOnItsService(t *testing.T) {
	m := fixture(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m.SetDrift(map[string]map[string]docker.ServiceSync{
		"web": {"api": docker.ServiceDrifted, "worker": docker.ServiceInSync},
	})
	if got := selectionOf(m); got != "service:worker" {
		t.Fatalf("expected the cursor to stay on worker across a drift update, got %s", got)
	}
}

// A filter can leave nothing but a header visible: "Services (2)" matches
// "services" while no service name does. There is then no selectable row at
// all, which must be reported as no selection rather than crash or lie.
func TestServicesModel_FilterMatchingOnlyAHeaderSelectsNothing(t *testing.T) {
	m := fixture(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: '%'})
	for _, r := range "services (2)" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if rows := m.table.FilteredRows(); len(rows) != 1 {
		t.Fatalf("precondition: expected only the Services header to match, got %d rows", len(rows))
	}
	if m.SelectedService() != nil || m.SelectedNetwork() != nil || m.SelectedVolume() != nil {
		t.Fatalf("expected no selection when only a header matches, got %s", selectionOf(m))
	}
}

// An empty view has nothing to select and must not panic on the way there.
func TestServicesModel_EmptyViewSelectsNothing(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices(nil, nil, nil, "web")
	if got := selectionOf(m); got != "none" {
		t.Fatalf("expected an empty view to select nothing, got %s", got)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := selectionOf(m); got != "none" {
		t.Fatalf("expected navigating an empty view to select nothing, got %s", got)
	}
}

// The cursor skip has to scroll as well as move, and sorting is what proves
// it: f1 reorders the rows under the cursor, so the skip jumps to wherever
// the nearest resource row now is, a jump, not a step, and the inner
// table's own SetCursor moves the cursor without moving the viewport. On a
// short terminal that leaves the selection highlighted off screen: no
// highlight anywhere on screen while every key acts on an invisible row.
func TestServicesModel_SortKeepsTheSelectionOnScreen(t *testing.T) {
	services := make([]docker.ComposeService, 5)
	for i := range services {
		services[i] = docker.ComposeService{Project: "web", Name: fmt.Sprintf("svc%02d", i)}
	}
	// Height 5 leaves the table one row tall, which a tmux split or a
	// 24-line terminal with the header shown produces.
	for _, height := range []int{5, 6, 8, 13} {
		m := NewServicesModel()
		m.SetSize(120, height)
		m.SetServices(services,
			[]docker.ComposeNetwork{{Name: "net00"}, {Name: "net01"}},
			[]docker.ComposeVolume{{Name: "vol00"}, {Name: "vol01"}, {Name: "vol02"}},
			"web")

		for sort := range 9 {
			m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
			selection := selectionOf(m)
			if strings.HasPrefix(selection, "unselectable") {
				t.Fatalf("height %d, sort %d: cursor on an unselectable row", height, sort+1)
			}
			name := selection[strings.Index(selection, ":")+1:]
			if !strings.Contains(highlightedLine(m), name) {
				t.Fatalf("height %d, sort %d: %s is selected but not highlighted on screen:\n%s",
					height, sort+1, selection, ansi.Strip(m.View()))
			}
		}
	}
}

// Walking to the bottom of a list taller than the viewport keeps the
// selection on screen too, section boundaries included.
func TestServicesModel_WalkingToTheBottomStaysOnScreen(t *testing.T) {
	services := make([]docker.ComposeService, 15)
	for i := range services {
		services[i] = docker.ComposeService{Project: "web", Name: fmt.Sprintf("svc%02d", i)}
	}
	for _, height := range []int{6, 13, 15, 17, 21, 29} {
		m := NewServicesModel()
		m.SetSize(120, height)
		m.SetServices(services,
			[]docker.ComposeNetwork{{Name: "web_default"}, {Name: "web_back"}},
			[]docker.ComposeVolume{{Name: "web_data"}},
			"web")

		for range len(services) + 8 {
			m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		}
		if got := selectionOf(m); got != "volume:web_data" {
			t.Fatalf("height %d: expected the last row selected, got %s", height, got)
		}
		if !strings.Contains(highlightedLine(m), "web_data") {
			t.Fatalf("height %d: the selected row is not highlighted on screen:\n%s",
				height, ansi.Strip(m.View()))
		}
	}
}

// A rebuild has no direction of travel. When a service disappears from above
// the cursor, every row below shifts up by one and a header lands under the
// cursor: the row the user was on is the one before it, so the skip has to
// look up, not down.
func TestServicesModel_ServiceRemovedAboveTheCursorKeepsTheUsersRow(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices(
		[]docker.ComposeService{
			{Project: "web", Name: "api"},
			{Project: "web", Name: "worker"},
		},
		[]docker.ComposeNetwork{{Name: "web_default"}, {Name: "web_back"}},
		nil, "web")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := selectionOf(m); got != "service:worker" {
		t.Fatalf("precondition: expected worker selected, got %s", got)
	}

	// api's containers are gone; the reload drops it.
	m.SetServices(
		[]docker.ComposeService{{Project: "web", Name: "worker"}},
		[]docker.ComposeNetwork{{Name: "web_default"}, {Name: "web_back"}},
		nil, "web")
	if got := selectionOf(m); got != "service:worker" {
		t.Fatalf("expected the cursor to stay on worker, got %s", got)
	}
}

// Entering a project switches the view before its resources arrive. The
// previous project's rows must not be left on screen under the new
// project's title, where a key would act on them.
func TestServicesModel_SetProjectClearsThePreviousProject(t *testing.T) {
	m := fixture(t)
	if selectionOf(m) != "service:api" {
		t.Fatalf("precondition: expected a selection, got %s", selectionOf(m))
	}

	m.SetProject("other")
	if got := selectionOf(m); got != "none" {
		t.Fatalf("expected no selection after switching project, got %s", got)
	}
	if m.table.RowCount() != 0 {
		t.Fatalf("expected no rows after switching project, got %d", m.table.RowCount())
	}
	if !strings.Contains(ansi.Strip(m.View()), "other") {
		t.Fatalf("expected the new project in the title, got:\n%s", ansi.Strip(m.View()))
	}
}

// A row type the Selected* accessors cannot resolve must not be a cursor
// target: that is the bug this file exists to prevent, one row type removed.
// Selectability is derived from the same resolution the accessors use, so a
// row added later is unselectable until it resolves to something.
type unknownRow struct{}

func (unknownRow) Columns() []string { return []string{"", "", "", "", "", "", "", ""} }
func (unknownRow) ID() string        { return "unknown" }

func TestSelectableRow_OnlyResolvableRowsAreSelectable(t *testing.T) {
	if selectableRow(unknownRow{}) {
		t.Fatal("a row the accessors cannot resolve must not be selectable")
	}
	if selectableRow(newSectionRow("Services", 1)) {
		t.Fatal("a section header must not be selectable")
	}
	for _, row := range []appui.TableRow{
		newServiceRow(docker.ComposeService{Name: "api"}, docker.ServiceInSync),
		newNetworkRow(docker.ComposeNetwork{Name: "net"}),
		newVolumeRow(docker.ComposeVolume{Name: "vol"}),
	} {
		if !selectableRow(row) {
			t.Fatalf("expected %T to be selectable", row)
		}
	}
}

// The cursor follows the row, not the index. A service appearing or
// disappearing above the cursor shifts every row below it by one, and
// compose reloads on every container event, so an index-keyed cursor slides
// onto a neighbour while the user is looking at it, including onto another
// resource, where nothing looks wrong until the next keypress acts on it.
func TestServicesModel_ReloadFollowsTheSelectedRow(t *testing.T) {
	services := []docker.ComposeService{
		{Project: "web", Name: "api"},
		{Project: "web", Name: "worker"},
	}
	networks := []docker.ComposeNetwork{{Name: "net00"}, {Name: "net01"}}

	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices(services, networks, nil, "web")
	for range 2 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if got := selectionOf(m); got != "network:net00" {
		t.Fatalf("precondition: expected net00 selected, got %s", got)
	}

	// api's containers are gone: every row below it shifts up by one.
	m.SetServices(services[1:], networks, nil, "web")
	if got := selectionOf(m); got != "network:net00" {
		t.Fatalf("expected the cursor to follow net00, got %s", got)
	}

	// api comes back: everything shifts down again.
	m.SetServices(services, networks, nil, "web")
	if got := selectionOf(m); got != "network:net00" {
		t.Fatalf("expected the cursor to still follow net00, got %s", got)
	}
}

// The filter was typed for a project. Carrying it into the next one hides
// rows the user never filtered, usually all of them, since service names
// rarely match across projects, and an empty list reads as an empty
// project.
func TestServicesModel_SetProjectClearsTheFilter(t *testing.T) {
	m := fixture(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: '%'})
	for _, r := range "api" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if m.table.FilterText() == "" {
		t.Fatal("precondition: expected an active filter")
	}

	m.SetProject("other")
	if got := m.table.FilterText(); got != "" {
		t.Fatalf("expected the filter to be cleared with the project, got %q", got)
	}
	if m.FilterActive() {
		t.Fatal("expected the filter input to be closed with the project")
	}

	m.SetServices([]docker.ComposeService{{Project: "other", Name: "db"}}, nil, nil, "other")
	if got := selectionOf(m); got != "service:db" {
		t.Fatalf("expected the new project's service to be selected, got %s", got)
	}
}

// A filter that matches only a section header leaves the cursor on it,
// there is nothing else to select. When a reload then brings in a row that
// does match, the cursor has to move onto it: following the header by ID
// would put the cursor straight back on a row no key can act on, which is
// the bug this file exists to prevent.
func TestServicesModel_ReloadUnderAHeaderOnlyFilterSelectsTheNewRow(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices([]docker.ComposeService{{Project: "web", Name: "api"}},
		nil, []docker.ComposeVolume{{Name: "cache"}}, "web")

	m, _ = m.Update(tea.KeyPressMsg{Code: '%'})
	for _, r := range "volumes" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if m.SelectedService() != nil || m.SelectedNetwork() != nil || m.SelectedVolume() != nil {
		t.Fatalf("precondition: expected only the header to match, got %s", selectionOf(m))
	}

	// A compose up creates a volume whose name matches the filter.
	m.SetServices([]docker.ComposeService{{Project: "web", Name: "api"}}, nil,
		[]docker.ComposeVolume{{Name: "cache"}, {Name: "web_volumes_data"}}, "web")

	if got := selectionOf(m); got != "volume:web_volumes_data" {
		t.Fatalf("expected the matching row to be selected, got %s", got)
	}
}

// Sorting reorders every row, so the cursor's index means nothing
// afterwards. Before, f1 slid the selection onto whatever landed under that
// index, often a different kind of resource, so the next enter inspected a
// network the user never chose.
func TestServicesModel_SortFollowsTheSelectedRow(t *testing.T) {
	m := fixture(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := selectionOf(m); got != "service:worker" {
		t.Fatalf("precondition: expected worker selected, got %s", got)
	}

	for sort := range 8 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
		if got := selectionOf(m); got != "service:worker" {
			t.Fatalf("sort %d: expected the cursor to stay on worker, got %s", sort+1, got)
		}
	}
}

// Typing a filter removes rows from the middle of the list, so by index the
// selection slides onto a neighbour while the user is looking at it.
func TestServicesModel_FilterFollowsTheSelectedRow(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	m.SetServices([]docker.ComposeService{
		{Project: "web", Name: "api"},
		{Project: "web", Name: "cache"},
		{Project: "web", Name: "worker"},
	}, []docker.ComposeNetwork{{Name: "web_default"}}, nil, "web")

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := selectionOf(m); got != "service:cache" {
		t.Fatalf("precondition: expected cache selected, got %s", got)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: '%'})
	for _, r := range "c" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if got := selectionOf(m); got != "service:cache" {
		t.Fatalf("expected the cursor to stay on cache while filtering, got %s", got)
	}
	// Backspacing back to the full list keeps it there too.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if got := selectionOf(m); got != "service:cache" {
		t.Fatalf("expected the cursor to stay on cache after backspace, got %s", got)
	}
}

// Row IDs are prefixed per kind, so a service whose name looks like another
// kind's ID cannot steal the cursor from it. Service names come from a
// container label, which is user-controlled.
func TestServicesModel_ARowIDCannotCollideAcrossKinds(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	services := []docker.ComposeService{{Project: "web", Name: "net:api"}}
	networks := []docker.ComposeNetwork{{Name: "api"}}
	m.SetServices(services, networks, nil, "web")

	// Select the network, then reload: the service named "net:api" must not
	// take the cursor.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := selectionOf(m); got != "network:api" {
		t.Fatalf("precondition: expected the network selected, got %s", got)
	}
	m.SetServices(services, networks, nil, "web")
	if got := selectionOf(m); got != "network:api" {
		t.Fatalf("expected the cursor to stay on the network, got %s", got)
	}
}

// pgup and pgdown are navigation like any other, so the header skip has to
// know their direction: pgup landing on a header must search upwards, or a
// page up from the middle of a section jumps forward past rows the user was
// paging towards.
func TestServicesModel_PageUpSkipsBackwards(t *testing.T) {
	services := make([]docker.ComposeService, 3)
	for i := range services {
		services[i] = docker.ComposeService{Project: "web", Name: fmt.Sprintf("svc%02d", i)}
	}
	// A short table, so a page is two rows: page up from the last network
	// lands exactly on the Networks header, with a service above it and a
	// network below, which is the only shape where the direction shows.
	m := NewServicesModel()
	m.SetSize(120, 6)
	m.SetServices(services, []docker.ComposeNetwork{{Name: "n0"}, {Name: "n1"}}, nil, "web")
	for range len(services) + 4 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if got := selectionOf(m); got != "network:n1" {
		t.Fatalf("precondition: expected the last network selected, got %s", got)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	if got := selectionOf(m); got != "service:svc02" {
		t.Fatalf("expected page up to carry on upwards onto the last service, got %s", got)
	}
	if !strings.Contains(highlightedLine(m), "svc02") {
		t.Fatalf("expected the selection on screen after page up, got:\n%s", ansi.Strip(m.View()))
	}
}

// When the selected row is gone, the row that takes its place is selected,
// the way deleting from any list works. The exception is a section header
// taking that place: the rebuild then prefers the row above, because
// everything below the removed row has shifted up and the nearest survivor
// is the one before the cursor's index, the row after the header belongs
// to a different section, and to a different kind of resource.
func TestServicesModel_RemovedRowFallsBackToTheRowAbove(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 40)
	services := []docker.ComposeService{
		{Project: "web", Name: "api"},
		{Project: "web", Name: "worker"},
	}
	networks := []docker.ComposeNetwork{{Name: "web_default"}}
	m.SetServices(services, networks, nil, "web")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := selectionOf(m); got != "service:worker" {
		t.Fatalf("precondition: expected worker selected, got %s", got)
	}

	// worker was the last row of its section, so the Networks header takes
	// its index.
	m.SetServices(services[:1], networks, nil, "web")
	if got := selectionOf(m); got != "service:api" {
		t.Fatalf("expected the row above the header, got %s", got)
	}

	// A row in the middle of a section is replaced by the one that takes
	// its place.
	m.SetServices(services, networks, nil, "web")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if got := selectionOf(m); got != "service:api" {
		t.Fatalf("precondition: expected api selected, got %s", got)
	}
	m.SetServices(services[1:], networks, nil, "web")
	if got := selectionOf(m); got != "service:worker" {
		t.Fatalf("expected the row that took its place, got %s", got)
	}
}

// The compose views' column set is the one the gutter bug was reported from
// , three proportional columns among five fixed ones, and the unit sweep
// in appui cannot reach it from another package. Sweep it here: at every
// width, the cell before each column boundary must be a space.
func TestComposeViews_EveryColumnKeepsItsGutterAtEveryWidth(t *testing.T) {
	services := NewServicesModel()
	projects := NewProjectsModel()
	sets := map[string]appui.TableModel{
		"services": services.table,
		"projects": projects.table,
	}
	for name, tbl := range sets {
		t.Run(name, func(t *testing.T) {
			cells := make([]string, 8)
			for i := range cells {
				cells[i] = strings.Repeat(string(rune('a'+i)), 60)
			}
			for width := 20; width <= 210; width++ {
				table := tbl
				table.SetSize(width, 12)
				table.SetRows([]appui.TableRow{sweepRow(cells)})

				lines := strings.Split(table.View(), "\n")
				if len(lines) < 2 {
					t.Fatalf("width %d: expected a header and a row", width)
				}
				boundary := 0
				for i := 0; i < len(cells)-1; i++ {
					boundary += table.ColumnWidth(i)
					if boundary >= width {
						break
					}
					if width := table.ColumnWidth(i); width < 2 {
						// One cell holds the ellipsis, with nothing left
						// for a gutter: the allocator prefers that to
						// overflowing the table. See
						// TestTableModel_FittingIsNeverTradedForSpacing.
						continue
					}
					for row, line := range lines[:2] {
						runes := []rune(ansi.Strip(line))
						if len(runes) < boundary {
							t.Fatalf("%s width %d: line %d is %d cells, expected %d",
								name, width, row, len(runes), boundary)
						}
						if runes[boundary-1] != ' ' {
							t.Fatalf("%s width %d: column %d butts column %d on line %d:\n%s",
								name, width, i, i+1, row, ansi.Strip(line))
						}
					}
				}
			}
		})
	}
}

// sweepRow is a row of pre-set cells for the width sweep.
type sweepRow []string

func (r sweepRow) Columns() []string { return r }
func (r sweepRow) ID() string        { return "sweep" }
