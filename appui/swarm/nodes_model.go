package swarm

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/docker/docker/api/types/swarm"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

// nodeRow wraps a swarm node as a TableRow.
type nodeRow struct {
	node    swarm.Node
	columns []string
}

func newNodeRow(n swarm.Node) nodeRow {
	role := string(n.Spec.Role)
	status := string(n.Status.State)
	availability := string(n.Spec.Availability)
	hostname := n.Description.Hostname
	cpu := fmt.Sprintf("%d", n.Description.Resources.NanoCPUs/1e9)
	mem := fmt.Sprintf("%d MB", n.Description.Resources.MemoryBytes/(1024*1024))
	return nodeRow{
		node: n,
		columns: []string{
			docker.TruncateID(n.ID), hostname, role, availability, status, cpu, mem,
		},
	}
}

func (r nodeRow) Columns() []string { return r.columns }
func (r nodeRow) ID() string        { return r.node.ID }

// NodesLoadedMsg carries the loaded nodes.
type NodesLoadedMsg struct {
	Nodes []swarm.Node
}

// NodesModel is the swarm nodes list view.
type NodesModel struct {
	table  appui.TableModel
	filter appui.FilterInputModel
	daemon docker.ContainerDaemon
}

// NewNodesModel creates a nodes list model.
func NewNodesModel() NodesModel {
	columns := []appui.Column{
		{Title: "ID", Width: appui.IDColumnWidth, Fixed: true},
		{Title: "HOSTNAME"},
		{Title: "ROLE", Width: 10, Fixed: true},
		{Title: "AVAILABILITY", Width: 14, Fixed: true},
		{Title: "STATUS", Width: 10, Fixed: true},
		{Title: "CPU", Width: 6, Fixed: true},
		{Title: "MEMORY", Width: 10, Fixed: true},
	}
	return NodesModel{
		table:  appui.NewTableModel(columns),
		filter: appui.NewFilterInputModel(),
	}
}

// SetDaemon sets the Docker daemon reference.
func (m *NodesModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// SetSize updates the table dimensions.
func (m *NodesModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-1-filterH)
	m.filter.SetWidth(w)
}

// SetNodes replaces the node list.
func (m *NodesModel) SetNodes(nodes []swarm.Node) {
	rows := make([]appui.TableRow, len(nodes))
	for i, n := range nodes {
		rows[i] = newNodeRow(n)
	}
	m.table.SetRows(rows)
}

// SelectedNode returns the node under the cursor, or nil.
func (m NodesModel) SelectedNode() *swarm.Node {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if nr, ok := row.(nodeRow); ok {
		return &nr.node
	}
	return nil
}

// Update handles key events.
func (m NodesModel) Update(msg tea.Msg) (NodesModel, tea.Cmd) {
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

// View renders the nodes list.
func (m NodesModel) View() string {
	header := m.widgetHeader()
	tableView := m.table.View()
	result := header + "\n" + tableView
	if filterView := m.filter.View(); filterView != "" {
		result += "\n" + filterView
	}
	return result
}

func (m NodesModel) widgetHeader() string {
	return appui.RenderWidgetHeader(appui.WidgetHeaderOpts{
		Icon:     "🖥️",
		Title:    "Nodes",
		Total:    m.table.TotalRowCount(),
		Filtered: m.table.TotalRowCount(),
		Width:    m.table.Width(),
		Accent:   appui.DryTheme.Success,
	})
}
