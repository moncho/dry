package compose

import (
	"fmt"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

// projectHeaderRow is a group header for a compose project.
type projectHeaderRow struct {
	project docker.ComposeProject
	columns []string
}

func newProjectHeaderRow(p docker.ComposeProject) projectHeaderRow {
	name := appui.ColorFg(p.Name, appui.DryTheme.Info)
	return projectHeaderRow{
		project: p,
		columns: []string{
			name,
			fmt.Sprintf("%d", p.Containers),
			fmt.Sprintf("%d", p.Running),
			fmt.Sprintf("%d", p.Exited),
			colorStatus(p.Status),
			"",
			"",
			"",
		},
	}
}

func (r projectHeaderRow) Columns() []string { return r.columns }

// ID prefixes the project name the way the services view prefixes its row
// kinds: project and service names both come from user-settable container
// labels, so an unprefixed project called "web/api" is the same string as
// project web's service api, and SelectRowByID cannot tell them apart.
func (r projectHeaderRow) ID() string { return "project:" + r.project.Name }

// serviceDetailRow is an indented row for a service within a project.
type serviceDetailRow struct {
	service     docker.ComposeService
	projectName string
	columns     []string
}

func newServiceDetailRow(s docker.ComposeService, sync docker.ServiceSync) serviceDetailRow {
	return serviceDetailRow{
		service:     s,
		projectName: s.Project,
		columns: []string{
			"  " + s.Name,
			fmt.Sprintf("%d", s.Containers),
			fmt.Sprintf("%d", s.Running),
			fmt.Sprintf("%d", s.Exited),
			s.Image,
			colorHealth(s.Health),
			colorSync(sync),
			s.Ports,
		},
	}
}

// colorStatus renders a project's lifecycle status. It shares a column with
// the services' images — a project row has no image and a service row has no
// project status, the same way the services view shares IMAGE/DRIVER between
// its row types — so the status costs no other column a single character of
// width. It is deliberately distinct from SYNC: STATUS is per-project
// lifecycle (does this project have running containers at all), SYNC is
// per-service drift against the compose file.
//
// A project with no computed status renders nothing rather than guessing at
// one; the empty ProjectStatus is what a project loaded before status
// existed, or one built by a caller that does not set it, carries.
func colorStatus(s docker.ProjectStatus) string {
	switch s {
	case docker.ProjectRunning:
		return appui.ColorFg(string(s), appui.DryTheme.Success)
	case docker.ProjectStopped:
		return appui.ColorFg(string(s), appui.DryTheme.Warning)
	case docker.ProjectNotCreated:
		return appui.ColorFg(string(s), appui.DryTheme.FgMuted)
	}
	return ""
}

// colorHealth applies theme colors to health status text.
func colorHealth(h string) string {
	switch h {
	case "unhealthy":
		return appui.ColorFg(h, appui.DryTheme.Error)
	case "healthy":
		return appui.ColorFg(h, appui.DryTheme.Success)
	case "starting":
		return appui.ColorFg(h, appui.DryTheme.Warning)
	default:
		return h
	}
}

// colorSync colors the sync status: drift is a warning, not an error. The
// rendered labels are short so the fixed-width SYNC column does not steal
// space from the flexible columns (IMAGE, PORTS); the ServiceSync constant
// values stay the spec's full words ("in sync", "drifted", "not created"),
// only the on-screen text is abbreviated. A project row's STATUS column
// spells "not created" in full, since it is a flexible column with the
// room for it.
func colorSync(s docker.ServiceSync) string {
	switch s {
	case docker.ServiceDrifted:
		return appui.ColorFg("drift", appui.DryTheme.Warning)
	case docker.ServiceNotCreated:
		// "absent", not "none": HEALTH, the column immediately to the
		// left, renders "none" for a container with no healthcheck, and
		// this is the first release in which this label reaches a screen
		// at all, so there is no reading of "none" here to preserve.
		return appui.ColorFg("absent", appui.DryTheme.FgMuted)
	case docker.ServiceInSync:
		return appui.ColorFg("ok", appui.DryTheme.Success)
	}
	return ""
}

func (r serviceDetailRow) Columns() []string { return r.columns }

// ID is prefixed by kind and joined by a NUL, because both halves come from
// user-settable container labels. Joined by a slash, project "a" service
// "b/c" and project "a/b" service "c" are the same string, and nothing
// parses this ID: the project comes off the row itself in selection().
func (r serviceDetailRow) ID() string {
	return "service:" + r.projectName + "\x00" + r.service.Name
}

// ProjectsLoadedMsg carries the loaded compose projects with services.
type ProjectsLoadedMsg struct {
	Projects []docker.ProjectWithServices
}

// ProjectsModel is the Compose projects list view.
type ProjectsModel struct {
	table    appui.TableModel
	filter   appui.FilterInputModel
	projects []docker.ProjectWithServices
	drift    map[string]map[string]docker.ServiceSync
	// loaded records that a project list has arrived, so an empty view can
	// say whether it is waiting or whether there is nothing to show.
	loaded bool
}

// NewProjectsModel creates a compose projects list model.
func NewProjectsModel() ProjectsModel {
	columns := []appui.Column{
		{Title: "NAME"},
		{Title: "CONTAINERS", Width: 12, Fixed: true},
		{Title: "RUNNING", Width: 10, Fixed: true},
		{Title: "EXITED", Width: 10, Fixed: true},
		{Title: "STATUS/IMAGE"},
		{Title: "HEALTH", Width: 12, Fixed: true},
		{Title: "SYNC", Width: 7, Fixed: true},
		{Title: "PORTS"},
	}
	m := ProjectsModel{
		table:  appui.NewTableModel(columns),
		filter: appui.NewFilterInputModel(),
	}
	m.syncEmptyMessage()
	return m
}

// FilterActive returns true when the filter input is active.
func (m ProjectsModel) FilterActive() bool { return m.filter.Active() }

// Filtered reports whether a filter is narrowing the list, open or not. It
// tells "no compose projects" apart from "the filter is hiding them", which
// is what the keys need to name a reason rather than doing nothing.
func (m ProjectsModel) Filtered() bool { return m.table.FilterText() != "" }

// SetFilter narrows the list, for tests that need a filter without driving
// the input; the view itself pushes the input's value in Update. Both go
// through applyFilter, so what the tests exercise is what the view runs.
func (m *ProjectsModel) SetFilter(pattern string) { m.applyFilter(pattern) }

// applyFilter narrows the list and keeps the cursor on the row it was on.
// Narrowing changes which rows are visible, so the cursor's index means
// something else afterwards.
func (m *ProjectsModel) applyFilter(pattern string) {
	selected, project := m.selection()
	m.table.SetFilter(pattern)
	m.restoreSelection(selected, project)
	m.syncEmptyMessage()
}

// ServiceRowCount is how many service rows a project has before filtering:
// the ones with containers plus the ones the compose file defines and no
// container exists for. The workspace panel sits beside the list, so a
// count taken from containers alone contradicts what the reader can see.
func (m ProjectsModel) ServiceRowCount(project string) int {
	for _, pws := range m.projects {
		if pws.Project.Name != project {
			continue
		}
		return len(pws.Services) + len(notCreatedServices(project, pws.Services, m.drift[project]))
	}
	return 0
}

// ProjectCount is how many projects the model holds. Zero covers both a list
// that has not loaded and a host with no compose projects at all; the caller
// in composeServiceDrift tells them apart because it is looking at a project
// it has already entered, so its list cannot legitimately be empty.
func (m ProjectsModel) ProjectCount() int { return len(m.projects) }

// SetSize updates the table dimensions.
func (m *ProjectsModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-2-filterH)
	m.filter.SetWidth(w)
}

// SetProjects replaces the project list with interleaved project+service rows.
func (m *ProjectsModel) SetProjects(projects []docker.ProjectWithServices) {
	m.projects = projects
	m.loaded = true
	m.refreshRows()
}

// SetDrift records per-service sync status, keyed by project then service.
func (m *ProjectsModel) SetDrift(drift map[string]map[string]docker.ServiceSync) {
	m.drift = drift
	m.refreshRows()
}

// refreshRows rebuilds the interleaved project+service rows from the current
// project list and drift status.
func (m *ProjectsModel) refreshRows() {
	// Ascending, the one direction any dry table sorts in.
	field, asc := m.table.SortField(), true
	var rows []appui.TableRow
	for _, pws := range m.sortedProjects(field, asc) {
		rows = append(rows, newProjectHeaderRow(pws.Project))
		sync := m.drift[pws.Project.Name]
		services := mergeByName(pws.Services, notCreatedServices(pws.Project.Name, pws.Services, sync))
		details := make([]appui.TableRow, 0, len(services))
		for _, svc := range services {
			details = append(details, newServiceDetailRow(svc, sync[svc.Name]))
		}
		sort.SliceStable(details, func(i, j int) bool {
			return appui.CompareRowsByColumn(details[i], details[j], field, asc)
		})
		rows = append(rows, details...)
	}
	selected, project := m.selection()
	m.table.SetRows(rows)
	m.restoreSelection(selected, project)
	m.syncEmptyMessage()
}

// selection is the row under the cursor and the project it belongs to. The
// project comes off the row rather than out of its ID, so nothing outside
// the row types has to know how an ID is built.
func (m ProjectsModel) selection() (id, project string) {
	row := m.table.SelectedRow()
	if row == nil {
		return "", ""
	}
	switch r := row.(type) {
	case projectHeaderRow:
		return r.ID(), r.project.Name
	case serviceDetailRow:
		return r.ID(), r.projectName
	}
	return row.ID(), ""
}

// restoreSelection puts the cursor back on the row it was on. Rows are
// rebuilt from scratch on every refresh, and drift can add or remove them on
// either side of the selection, so following the index moves the selection
// under the user's hands.
//
// When the row itself is gone it prefers, in order: the nearest remaining
// service of the same project, that project's own header, then any service
// row at all. It is a best effort and nothing more, since a filter can hide
// every row of that project, so no ordering here guarantees the cursor
// stays in the project it started in. Nothing destructive rests on it: the
// keys that act on the cursor's row all name their target in a prompt.
func (m *ProjectsModel) restoreSelection(id, project string) {
	if id == "" || m.table.SelectRowByID(id) {
		return
	}
	rows := m.table.FilteredRows()
	cursor := m.table.Cursor()
	nearest := -1
	for i, row := range rows {
		r, ok := row.(serviceDetailRow)
		if !ok || r.projectName != project {
			continue
		}
		if nearest < 0 || abs(i-cursor) < abs(nearest-cursor) {
			nearest = i
		}
	}
	if nearest >= 0 {
		m.table.SetCursor(nearest)
		return
	}
	if project != "" && m.table.SelectRowByID("project:"+project) {
		return
	}
	m.stepOffProjectHeader()
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// stepOffProjectHeader moves the cursor onto the nearest service row when it
// is resting on a header it was not put on deliberately. A header is a
// legitimate target here, so this runs only from restoreSelection, once the
// followed row and its project are both gone.
func (m *ProjectsModel) stepOffProjectHeader() {
	rows := m.table.FilteredRows()
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(rows) {
		return
	}
	if _, onHeader := rows[cursor].(projectHeaderRow); !onHeader {
		return
	}
	// Below first, then above: a forward-only walk settles on the header it
	// was asked to leave whenever every row below it is another header,
	// which is what a project with nothing running looks like.
	for i := cursor + 1; i < len(rows); i++ {
		if _, header := rows[i].(projectHeaderRow); !header {
			m.table.SetCursor(i)
			return
		}
	}
	for i := cursor - 1; i >= 0; i-- {
		if _, header := rows[i].(projectHeaderRow); !header {
			m.table.SetCursor(i)
			return
		}
	}
}

// syncEmptyMessage pushes the current reason into the table. Called from
// every state change rather than from View, which has a value receiver and
// would throw the update away.
func (m *ProjectsModel) syncEmptyMessage() { m.table.SetEmptyMessage(m.emptyMessage()) }

// emptyMessage says why there is nothing to show: blank space cannot tell a
// list that has not arrived from a host with no compose projects.
func (m ProjectsModel) emptyMessage() string {
	switch {
	case !m.loaded:
		return "Loading compose projects..."
	case m.table.FilterText() != "":
		return "Nothing here matches the filter"
	default:
		// Where dry looks, not what it failed to find: whether a compose
		// file exists here is not something this model knows. It fits the
		// 58 columns workspace mode leaves the list on a 100-column
		// terminal, and the half that says where to look is the half a
		// longer message loses.
		return "No compose projects: dry reads labels and this directory"
	}
}

// sortedProjects orders the project groups by the active sort field,
// comparing the header rows the way the table compares any other row: the
// groups are ordered by their headers and the services inside them
// separately, which is what the table's own flat sort cannot do. Sorted
// flat, it moves services between headers and leaves service rows above the
// first header. On the default field this is the order the daemon already
// returns projects in.
func (m ProjectsModel) sortedProjects(field int, asc bool) []docker.ProjectWithServices {
	if len(m.projects) < 2 {
		return m.projects
	}
	out := make([]docker.ProjectWithServices, len(m.projects))
	copy(out, m.projects)
	sort.SliceStable(out, func(i, j int) bool {
		return appui.CompareRowsByColumn(
			newProjectHeaderRow(out[i].Project), newProjectHeaderRow(out[j].Project), field, asc)
	})
	return out
}

// SelectedService returns the service under the cursor, or nil if on a project row.
func (m ProjectsModel) SelectedService() *docker.ComposeService {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if r, ok := row.(serviceDetailRow); ok {
		return &r.service
	}
	return nil
}

// SelectedProject returns the project under the cursor, or nil.
// If the cursor is on a service row, returns the parent project.
func (m ProjectsModel) SelectedProject() *docker.ComposeProject {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	switch r := row.(type) {
	case projectHeaderRow:
		return &r.project
	case serviceDetailRow:
		return m.ProjectByName(r.projectName)
	}
	return nil
}

// ProjectByName returns the project row matching name, or nil if not loaded.
// A service row must be brought up in the context of its own project's
// files, so callers resolve the service's project name back to the full
// docker.ComposeProject before invoking a compose command.
func (m ProjectsModel) ProjectByName(name string) *docker.ComposeProject {
	for i := range m.projects {
		if m.projects[i].Project.Name == name {
			return &m.projects[i].Project
		}
	}
	return nil
}

// Update handles key events.
func (m ProjectsModel) Update(msg tea.Msg) (ProjectsModel, tea.Cmd) {
	if m.filter.Active() {
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.applyFilter(m.filter.Value())
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f1":
			// SetSortField moves the indicator without sorting, and the
			// rebuild orders the groups and their services separately.
			// NextSort would sort the rows flat, which moves services
			// under other projects' headers. refreshRows follows the
			// selected row, whose index means nothing after a reorder.
			m.table.NextSortField()
			m.refreshRows()
			return m, nil
		case "f5":
			return m, nil
		case "%":
			cmd := m.filter.Activate()
			return m, cmd
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the projects list.
func (m ProjectsModel) View() string {
	header := m.widgetHeader()
	tableView := m.table.View()
	result := header + "\n" + tableView
	if filterView := m.filter.View(); filterView != "" {
		result += "\n" + filterView
	}
	return result
}

// RefreshTableStyles re-applies theme styles to the inner table.
func (m *ProjectsModel) RefreshTableStyles() {
	m.table.RefreshStyles()
}

func (m ProjectsModel) widgetHeader() string {
	total := len(m.projects)
	filtered := total
	if m.table.FilterText() != "" {
		filtered = m.countFilteredProjects()
	}
	return appui.RenderWidgetHeader(appui.WidgetHeaderOpts{
		Icon:     "\U0001f433",
		Title:    "Compose Projects",
		Total:    total,
		Filtered: filtered,
		Filter:   m.table.FilterText(),
		Width:    m.table.Width(),
		Accent:   appui.DryTheme.Info,
	})
}

// countFilteredProjects counts how many project header rows survive the current filter.
func (m ProjectsModel) countFilteredProjects() int {
	count := 0
	for _, row := range m.table.FilteredRows() {
		if _, ok := row.(projectHeaderRow); ok {
			count++
		}
	}
	return count
}
