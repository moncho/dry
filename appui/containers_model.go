package appui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
)

// containerRow wraps a Docker container as a TableRow.
type containerRow struct {
	container *docker.Container
	columns   []string
	running   bool
}

func newContainerRow(c *docker.Container) containerRow {
	cf := formatter.NewContainerFormatter(c, true)
	running := docker.IsContainerRunning(c)
	indicator := "\u25A3" // ▣ status symbol
	return containerRow{
		container: c,
		columns: []string{
			indicator, cf.ID(), cf.Image(), cf.Command(),
			cf.Status(), cf.Ports(), cf.Names(),
		},
		running: running,
	}
}

func (r containerRow) Columns() []string { return r.columns }
func (r containerRow) ID() string        { return r.container.ID }

// ContainersModel is the container list view sub-model.
type ContainersModel struct {
	table    TableModel
	filter   FilterInputModel
	daemon   docker.ContainerDaemon
	showAll  bool
	sortMode docker.SortMode
}

// NewContainersModel creates a container list model.
func NewContainersModel() ContainersModel {
	columns := []Column{
		{Title: "", Width: 2, Fixed: true},
		{Title: "CONTAINER", Width: IDColumnWidth, Fixed: true},
		{Title: "IMAGE"},
		{Title: "COMMAND"},
		{Title: "STATUS", Width: 18, Fixed: true},
		{Title: "PORTS"},
		{Title: "NAMES"},
	}
	return ContainersModel{
		table:    NewTableModel(columns),
		filter:   NewFilterInputModel(),
		sortMode: docker.SortByContainerID,
	}
}

// SetDaemon sets the Docker daemon reference.
func (m *ContainersModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// SetSize updates the table dimensions.
func (m *ContainersModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-1-filterH) // -1 for widget header
	m.filter.SetWidth(w)
}

// ShowAll returns the show-all state.
func (m ContainersModel) ShowAll() bool {
	return m.showAll
}

// SortMode returns the current sort mode.
func (m ContainersModel) SortMode() docker.SortMode {
	return m.sortMode
}

// SetContainers replaces the container list with new data.
func (m *ContainersModel) SetContainers(containers []*docker.Container) {
	rows := make([]TableRow, len(containers))
	for i, c := range containers {
		rows[i] = newContainerRow(c)
	}
	m.table.SetRows(rows)
}

// SelectedContainer returns the container under the cursor, or nil.
func (m ContainersModel) SelectedContainer() *docker.Container {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if cr, ok := row.(containerRow); ok {
		return cr.container
	}
	return nil
}

// Update handles container-list-specific key events.
func (m ContainersModel) Update(msg tea.Msg) (ContainersModel, tea.Cmd) {
	// When filter input is active, forward everything to it
	if m.filter.Active() {
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		// Apply filter text to table in real-time
		m.table.SetFilter(m.filter.Value())
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f1":
			m.nextSort()
			return m, nil
		case "f2":
			m.showAll = !m.showAll
			return m, nil // parent handles reload
		case "f5":
			return m, nil // parent handles reload
		case "%":
			cmd := m.filter.Activate()
			return m, cmd
		}
	}
	// Forward to table for navigation
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the container list.
func (m ContainersModel) View() string {
	header := m.widgetHeader()
	tableView := m.table.View()
	result := header + "\n" + tableView
	if filterView := m.filter.View(); filterView != "" {
		result += "\n" + filterView
	}
	return result
}

func (m ContainersModel) widgetHeader() string {
	total := m.table.TotalRowCount()
	filtered := m.table.RowCount()
	filter := m.table.FilterText()

	title := fmt.Sprintf("Containers: %d", total)
	if total != filtered {
		title = fmt.Sprintf("Containers: %d (showing %d)", total, filtered)
	}
	if filter != "" {
		title += fmt.Sprintf(" | Filter: %s", filter)
	}

	style := lipgloss.NewStyle().Bold(true).Foreground(DryTheme.Key)
	return style.Render(title)
}

func (m *ContainersModel) nextSort() {
	m.sortMode++
	if m.sortMode > docker.SortByName {
		m.sortMode = docker.NoSort
	}
	m.table.NextSort()
}
