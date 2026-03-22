package appui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
)

// containerRow wraps a Docker container as a TableRow.
type containerRow struct {
	container *docker.Container
	columns   []string
}

func newContainerRow(c *docker.Container, compact bool) containerRow {
	cf := formatter.NewContainerFormatter(c, true)
	indicator := ColorFg("\u25A0", DryTheme.FgSubtle) // ■ stopped
	if docker.IsContainerRunning(c) {
		indicator = ColorFg("\u25B6", DryTheme.Key) // ▶ running
	}
	columns := []string{
		indicator, cf.ID(), cf.Image(), cf.Command(),
		cf.Status(), cf.Ports(), cf.Names(),
	}
	if compact {
		columns = []string{
			indicator, cf.ID(), cf.Image(),
			cf.Status(), cf.Names(),
		}
	}
	return containerRow{
		container: c,
		columns:   columns,
	}
}

func (r containerRow) Columns() []string { return r.columns }
func (r containerRow) ID() string        { return r.container.ID }

// ContainersModel is the container list view sub-model.
type ContainersModel struct {
	table    TableModel
	filter   FilterInputModel
	daemon   docker.ContainerDaemon
	rows     []*docker.Container
	showAll  bool
	sortMode docker.SortMode
	compact  bool
}

func containerColumns(compact bool) []Column {
	if compact {
		return []Column{
			{Title: "", Width: 2, Fixed: true},
			{Title: "CONTAINER", Width: IDColumnWidth, Fixed: true},
			{Title: "IMAGE"},
			{Title: "STATUS", Width: 18, Fixed: true},
			{Title: "NAMES"},
		}
	}
	return []Column{
		{Title: "", Width: 2, Fixed: true},
		{Title: "CONTAINER", Width: IDColumnWidth, Fixed: true},
		{Title: "IMAGE"},
		{Title: "COMMAND"},
		{Title: "STATUS", Width: 18, Fixed: true},
		{Title: "PORTS"},
		{Title: "NAMES"},
	}
}

// NewContainersModel creates a container list model.
func NewContainersModel() ContainersModel {
	return ContainersModel{
		table:    NewTableModel(containerColumns(false)),
		filter:   NewFilterInputModel(),
		sortMode: docker.SortByContainerID,
	}
}

// FilterActive returns true when the filter input is active.
func (m ContainersModel) FilterActive() bool { return m.filter.Active() }

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
	m.table.SetSize(w, h-2-filterH) // -2 for widget header + blank line
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

// SetCompact toggles the compact workspace column set.
func (m *ContainersModel) SetCompact(compact bool) {
	if m.compact == compact {
		return
	}
	m.compact = compact
	m.rebuildTable()
}

// SetContainers replaces the container list with new data.
func (m *ContainersModel) SetContainers(containers []*docker.Container) {
	m.rows = containers
	m.rebuildRows()
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
	return RenderWidgetHeader(WidgetHeaderOpts{
		Icon:     "🐳",
		Title:    "Containers",
		Total:    m.table.TotalRowCount(),
		Filtered: m.table.RowCount(),
		Filter:   m.table.FilterText(),
		Width:    m.table.Width(),
		Accent:   DryTheme.Info,
	})
}

// RefreshTableStyles re-applies theme styles to the inner table.
func (m *ContainersModel) RefreshTableStyles() {
	m.table.RefreshStyles()
}

func (m *ContainersModel) nextSort() {
	m.sortMode++
	if m.sortMode > docker.SortByName {
		m.sortMode = docker.NoSort
	}
	m.applySortIndicator()
}

func (m *ContainersModel) applySortIndicator() {
	col := -1
	switch m.sortMode {
	case docker.SortByContainerID:
		col = 1
	case docker.SortByImage:
		col = 2
	case docker.SortByStatus:
		if m.compact {
			col = 3
		} else {
			col = 4
		}
	case docker.SortByName:
		if m.compact {
			col = 4
		} else {
			col = 6
		}
	}
	m.table.SetSortField(col)
}

func (m *ContainersModel) rebuildRows() {
	rows := make([]TableRow, len(m.rows))
	for i, c := range m.rows {
		rows[i] = newContainerRow(c, m.compact)
	}
	m.table.SetRows(rows)
	m.applySortIndicator()
}

func (m *ContainersModel) rebuildTable() {
	width := m.table.width
	height := m.table.height
	filterText := m.filter.Value()

	m.table = NewTableModel(containerColumns(m.compact))
	if width > 0 && height > 0 {
		m.table.SetSize(width, height)
	}
	m.rebuildRows()
	if filterText != "" {
		m.table.SetFilter(filterText)
	}
}
