package appui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/docker/docker/api/types/network"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
)

// networkRow wraps a Docker network as a TableRow.
type networkRow struct {
	network network.Inspect
	columns []string
}

func newNetworkRow(n network.Inspect) networkRow {
	f := formatter.NewNetworkFormatter(n, true)
	return networkRow{
		network: n,
		columns: []string{
			f.ID(), f.Name(), f.Driver(), f.Containers(), f.Scope(), f.Subnet(),
		},
	}
}

func (r networkRow) Columns() []string { return r.columns }
func (r networkRow) ID() string        { return r.network.ID }

// NetworksLoadedMsg carries the loaded networks.
type NetworksLoadedMsg struct {
	Networks []network.Inspect
}

// NetworksModel is the networks list view sub-model.
type NetworksModel struct {
	table  TableModel
	filter FilterInputModel
	daemon docker.ContainerDaemon
}

// NewNetworksModel creates a networks list model.
func NewNetworksModel() NetworksModel {
	columns := []Column{
		{Title: "ID", Width: IDColumnWidth, Fixed: true},
		{Title: "NAME"},
		{Title: "DRIVER", Width: 12, Fixed: true},
		{Title: "CONTAINERS", Width: 12, Fixed: true},
		{Title: "SCOPE", Width: 8, Fixed: true},
		{Title: "SUBNET"},
	}
	return NetworksModel{
		table:  NewTableModel(columns),
		filter: NewFilterInputModel(),
	}
}

// SetDaemon sets the Docker daemon reference.
func (m *NetworksModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// SetSize updates the table dimensions.
func (m *NetworksModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-2-filterH)
	m.filter.SetWidth(w)
}

// SetNetworks replaces the network list.
func (m *NetworksModel) SetNetworks(networks []network.Inspect) {
	rows := make([]TableRow, len(networks))
	for i, n := range networks {
		rows[i] = newNetworkRow(n)
	}
	m.table.SetRows(rows)
}

// SelectedNetwork returns the network under the cursor, or nil.
func (m NetworksModel) SelectedNetwork() *network.Inspect {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if nr, ok := row.(networkRow); ok {
		return &nr.network
	}
	return nil
}

// Update handles network-list-specific key events.
func (m NetworksModel) Update(msg tea.Msg) (NetworksModel, tea.Cmd) {
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

// View renders the networks list.
func (m NetworksModel) View() string {
	header := m.widgetHeader()
	tableView := m.table.View()
	result := header + "\n" + tableView
	if filterView := m.filter.View(); filterView != "" {
		result += "\n" + filterView
	}
	return result
}

// RefreshTableStyles re-applies theme styles to the inner table.
func (m *NetworksModel) RefreshTableStyles() {
	m.table.RefreshStyles()
}

func (m NetworksModel) widgetHeader() string {
	return RenderWidgetHeader(WidgetHeaderOpts{
		Icon:     "🔗",
		Title:    "Networks",
		Total:    m.table.TotalRowCount(),
		Filtered: m.table.RowCount(),
		Filter:   m.table.FilterText(),
		Width:    m.table.Width(),
		Accent:   DryTheme.Tertiary,
	})
}
