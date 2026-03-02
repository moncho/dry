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

func newServiceDetailRow(s docker.ComposeService) serviceDetailRow {
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
			s.Ports,
		},
	}
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
	daemon   docker.ContainerDaemon
	projects []docker.ProjectWithServices
}

// NewProjectsModel creates a compose projects list model.
func NewProjectsModel() ProjectsModel {
	columns := []appui.Column{
		{Title: "NAME"},
		{Title: "CONTAINERS", Width: 12, Fixed: true},
		{Title: "RUNNING", Width: 10, Fixed: true},
		{Title: "EXITED", Width: 10, Fixed: true},
		{Title: "IMAGE"},
		{Title: "HEALTH", Width: 12, Fixed: true},
		{Title: "PORTS"},
	}
	return ProjectsModel{
		table:  appui.NewTableModel(columns),
		filter: appui.NewFilterInputModel(),
	}
}

// SetDaemon sets the Docker daemon reference.
func (m *ProjectsModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// FilterActive returns true when the filter input is active.
func (m ProjectsModel) FilterActive() bool { return m.filter.Active() }

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
	var rows []appui.TableRow
	for _, pws := range projects {
		rows = append(rows, newProjectHeaderRow(pws.Project))
		for _, svc := range pws.Services {
			rows = append(rows, newServiceDetailRow(svc))
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
		for i := range m.projects {
			if m.projects[i].Project.Name == r.projectName {
				return &m.projects[i].Project
			}
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
	return appui.RenderWidgetHeader(appui.WidgetHeaderOpts{
		Icon:     "\U0001f433",
		Title:    "Compose Projects",
		Total:    m.table.TotalRowCount(),
		Filtered: m.table.RowCount(),
		Filter:   m.table.FilterText(),
		Width:    m.table.Width(),
		Accent:   appui.DryTheme.Info,
	})
}
