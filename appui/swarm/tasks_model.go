package swarm

import (
	tea "charm.land/bubbletea/v2"
	"github.com/docker/docker/api/types/swarm"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
)

// taskRow wraps a swarm task as a TableRow.
type taskRow struct {
	task    swarm.Task
	columns []string
}

func newTaskRow(api docker.SwarmAPI, t swarm.Task) taskRow {
	if api == nil {
		return taskRow{
			task: t,
			columns: []string{
				docker.TruncateID(t.ID), "", "", "",
				string(t.DesiredState), string(t.Status.State), t.Status.Err,
			},
		}
	}
	ts := formatter.NewTaskStringer(api, t, true)
	return taskRow{
		task: t,
		columns: []string{
			ts.ID(), ts.Name(), ts.Image(), ts.NodeID(),
			ts.DesiredState(), ts.CurrentState(), ts.Error(),
		},
	}
}

func (r taskRow) Columns() []string { return r.columns }
func (r taskRow) ID() string        { return r.task.ID }

// TasksLoadedMsg carries the loaded tasks.
type TasksLoadedMsg struct {
	Tasks []swarm.Task
	Title string
}

// TasksModel is the swarm tasks list view.
type TasksModel struct {
	table  appui.TableModel
	filter appui.FilterInputModel
	daemon docker.ContainerDaemon
	title  string
}

// NewTasksModel creates a tasks list model.
func NewTasksModel() TasksModel {
	columns := []appui.Column{
		{Title: "ID", Width: appui.IDColumnWidth, Fixed: true},
		{Title: "NAME"},
		{Title: "IMAGE"},
		{Title: "NODE", Width: appui.IDColumnWidth, Fixed: true},
		{Title: "DESIRED", Width: 10, Fixed: true},
		{Title: "CURRENT", Width: 10, Fixed: true},
		{Title: "ERROR"},
	}
	return TasksModel{
		table:  appui.NewTableModel(columns),
		filter: appui.NewFilterInputModel(),
		title:  "Tasks",
	}
}

// FilterActive returns true when the filter input is active.
func (m TasksModel) FilterActive() bool { return m.filter.Active() }

// SetDaemon sets the Docker daemon reference.
func (m *TasksModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// SetSize updates the table dimensions.
func (m *TasksModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-2-filterH)
	m.filter.SetWidth(w)
}

// SetTasks replaces the task list.
func (m *TasksModel) SetTasks(tasks []swarm.Task, title string) {
	m.title = title
	rows := make([]appui.TableRow, len(tasks))
	for i, t := range tasks {
		rows[i] = newTaskRow(m.daemon, t)
	}
	m.table.SetRows(rows)
}

// Update handles key events.
func (m TasksModel) Update(msg tea.Msg) (TasksModel, tea.Cmd) {
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

// RefreshTableStyles re-applies theme styles to the inner table.
func (m *TasksModel) RefreshTableStyles() {
	m.table.RefreshStyles()
}

// View renders the tasks list.
func (m TasksModel) View() string {
	header := appui.RenderWidgetHeader(appui.WidgetHeaderOpts{
		Icon:     "📋",
		Title:    m.title,
		Total:    m.table.TotalRowCount(),
		Filtered: m.table.RowCount(),
		Filter:   m.table.FilterText(),
		Width:    m.table.Width(),
		Accent:   appui.DryTheme.Info,
	})
	result := header + "\n" + m.table.View()
	if filterView := m.filter.View(); filterView != "" {
		result += "\n" + filterView
	}
	return result
}
