package swarm

import (
	"fmt"

	"github.com/docker/docker/api/types/swarm"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

// serviceRow wraps a swarm service as a TableRow.
type serviceRow struct {
	service swarm.Service
	columns []string
}

func newServiceRow(s swarm.Service) serviceRow {
	var replicas string
	if s.Spec.Mode.Replicated != nil {
		replicas = fmt.Sprintf("%d", *s.Spec.Mode.Replicated.Replicas)
	} else {
		replicas = "global"
	}
	img := s.Spec.TaskTemplate.ContainerSpec.Image
	return serviceRow{
		service: s,
		columns: []string{
			docker.TruncateID(s.ID), s.Spec.Name, replicas, img,
		},
	}
}

func (r serviceRow) Columns() []string { return r.columns }
func (r serviceRow) ID() string        { return r.service.ID }

// ServicesLoadedMsg carries the loaded services.
type ServicesLoadedMsg struct {
	Services []swarm.Service
}

// ServicesModel is the swarm services list view.
type ServicesModel struct {
	table  appui.TableModel
	filter appui.FilterInputModel
	daemon docker.ContainerDaemon
}

// NewServicesModel creates a services list model.
func NewServicesModel() ServicesModel {
	columns := []appui.Column{
		{Title: "ID", Width: appui.IDColumnWidth, Fixed: true},
		{Title: "NAME"},
		{Title: "REPLICAS", Width: 10, Fixed: true},
		{Title: "IMAGE"},
	}
	return ServicesModel{
		table:  appui.NewTableModel(columns),
		filter: appui.NewFilterInputModel(),
	}
}

// SetDaemon sets the Docker daemon reference.
func (m *ServicesModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// SetSize updates the table dimensions.
func (m *ServicesModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-1-filterH)
	m.filter.SetWidth(w)
}

// SetServices replaces the service list.
func (m *ServicesModel) SetServices(services []swarm.Service) {
	rows := make([]appui.TableRow, len(services))
	for i, s := range services {
		rows[i] = newServiceRow(s)
	}
	m.table.SetRows(rows)
}

// SelectedService returns the service under the cursor, or nil.
func (m ServicesModel) SelectedService() *swarm.Service {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if sr, ok := row.(serviceRow); ok {
		return &sr.service
	}
	return nil
}

// Update handles key events.
func (m ServicesModel) Update(msg tea.Msg) (ServicesModel, tea.Cmd) {
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

// View renders the services list.
func (m ServicesModel) View() string {
	header := m.widgetHeader()
	tableView := m.table.View()
	result := header + "\n" + tableView
	if filterView := m.filter.View(); filterView != "" {
		result += "\n" + filterView
	}
	return result
}

func (m ServicesModel) widgetHeader() string {
	total := m.table.TotalRowCount()
	title := fmt.Sprintf("Services: %d", total)
	style := lipgloss.NewStyle().Bold(true).Foreground(appui.DryTheme.Key)
	return style.Render(title)
}
