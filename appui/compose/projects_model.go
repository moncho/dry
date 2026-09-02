package compose

import (
	"fmt"

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
func (r projectHeaderRow) ID() string        { return r.project.Name }

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
// only the on-screen text is abbreviated.
func colorSync(s docker.ServiceSync) string {
	switch s {
	case docker.ServiceDrifted:
		return appui.ColorFg("drift", appui.DryTheme.Warning)
	case docker.ServiceNotCreated:
		return appui.ColorFg("none", appui.DryTheme.FgMuted)
	case docker.ServiceInSync:
		return appui.ColorFg("ok", appui.DryTheme.Success)
	}
	return ""
}

func (r serviceDetailRow) Columns() []string { return r.columns }
func (r serviceDetailRow) ID() string        { return r.projectName + "/" + r.service.Name }

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
	return ProjectsModel{
		table:  appui.NewTableModel(columns),
		filter: appui.NewFilterInputModel(),
	}
}

// FilterActive returns true when the filter input is active.
func (m ProjectsModel) FilterActive() bool { return m.filter.Active() }

// Filtered reports whether a filter is narrowing the list, open or not. It
// tells "no compose projects" apart from "the filter is hiding them", which
// is what the keys need to name a reason rather than doing nothing.
func (m ProjectsModel) Filtered() bool { return m.table.FilterText() != "" }

// SetFilter narrows the list, for tests that need a filter without driving
// the input; the view itself pushes the input's value in Update.
func (m *ProjectsModel) SetFilter(pattern string) { m.table.SetFilter(pattern) }

// ProjectCount is how many projects the model holds. Zero covers both a list
// that has not loaded and a host with no compose projects at all.
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
	var rows []appui.TableRow
	for _, pws := range m.projects {
		rows = append(rows, newProjectHeaderRow(pws.Project))
		for _, svc := range pws.Services {
			sync := m.drift[pws.Project.Name][svc.Name]
			rows = append(rows, newServiceDetailRow(svc, sync))
		}
	}
	m.table.SetRows(rows)
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
		m.table.SetFilter(m.filter.Value())
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f1":
			m.table.NextSort()
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
