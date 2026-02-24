package swarm

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

// stackRow wraps a Docker stack as a TableRow.
type stackRow struct {
	stack   docker.Stack
	columns []string
}

func newStackRow(s docker.Stack) stackRow {
	return stackRow{
		stack: s,
		columns: []string{
			s.Name,
			fmt.Sprintf("%d", s.Services),
			fmt.Sprintf("%d", s.Networks),
			fmt.Sprintf("%d", s.Configs),
			fmt.Sprintf("%d", s.Secrets),
		},
	}
}

func (r stackRow) Columns() []string { return r.columns }
func (r stackRow) ID() string        { return r.stack.Name }

// StacksLoadedMsg carries the loaded stacks.
type StacksLoadedMsg struct {
	Stacks []docker.Stack
}

// StacksModel is the swarm stacks list view.
type StacksModel struct {
	table  appui.TableModel
	filter appui.FilterInputModel
	daemon docker.ContainerDaemon
}

// NewStacksModel creates a stacks list model.
func NewStacksModel() StacksModel {
	columns := []appui.Column{
		{Title: "NAME"},
		{Title: "SERVICES", Width: 10, Fixed: true},
		{Title: "NETWORKS", Width: 10, Fixed: true},
		{Title: "CONFIGS", Width: 10, Fixed: true},
		{Title: "SECRETS", Width: 10, Fixed: true},
	}
	return StacksModel{
		table:  appui.NewTableModel(columns),
		filter: appui.NewFilterInputModel(),
	}
}

// SetDaemon sets the Docker daemon reference.
func (m *StacksModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// SetSize updates the table dimensions.
func (m *StacksModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-1-filterH)
	m.filter.SetWidth(w)
}

// SetStacks replaces the stack list.
func (m *StacksModel) SetStacks(stacks []docker.Stack) {
	rows := make([]appui.TableRow, len(stacks))
	for i, s := range stacks {
		rows[i] = newStackRow(s)
	}
	m.table.SetRows(rows)
}

// SelectedStack returns the stack under the cursor, or nil.
func (m StacksModel) SelectedStack() *docker.Stack {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if sr, ok := row.(stackRow); ok {
		return &sr.stack
	}
	return nil
}

// Update handles key events.
func (m StacksModel) Update(msg tea.Msg) (StacksModel, tea.Cmd) {
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

// View renders the stacks list.
func (m StacksModel) View() string {
	header := m.widgetHeader()
	tableView := m.table.View()
	result := header + "\n" + tableView
	if filterView := m.filter.View(); filterView != "" {
		result += "\n" + filterView
	}
	return result
}

func (m StacksModel) widgetHeader() string {
	total := m.table.TotalRowCount()
	title := fmt.Sprintf("Stacks: %d", total)
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	return style.Render(title)
}
