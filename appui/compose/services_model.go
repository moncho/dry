package compose

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

// serviceRow wraps a ComposeService as a TableRow.
type serviceRow struct {
	service docker.ComposeService
	columns []string
}

func newServiceRow(s docker.ComposeService) serviceRow {
	return serviceRow{
		service: s,
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

func (r serviceRow) Columns() []string { return r.columns }
func (r serviceRow) ID() string        { return r.service.Name }

// sectionRow is a visual group header separating resource types.
type sectionRow struct {
	label   string
	columns []string
}

func newSectionRow(label string, count int) sectionRow {
	title := appui.ColorFg(fmt.Sprintf("%s (%d)", label, count), appui.DryTheme.Info)
	return sectionRow{
		label:   label,
		columns: []string{title, "", "", "", "", "", ""},
	}
}

func (r sectionRow) Columns() []string { return r.columns }
func (r sectionRow) ID() string        { return "section:" + r.label }

// networkRow wraps a ComposeNetwork as a TableRow.
type networkRow struct {
	network docker.ComposeNetwork
	columns []string
}

func newNetworkRow(n docker.ComposeNetwork) networkRow {
	return networkRow{
		network: n,
		columns: []string{"  " + n.Name, "", "", "", n.Driver, n.Scope, ""},
	}
}

func (r networkRow) Columns() []string { return r.columns }
func (r networkRow) ID() string        { return "net:" + r.network.Name }

// volumeRow wraps a ComposeVolume as a TableRow.
type volumeRow struct {
	volume  docker.ComposeVolume
	columns []string
}

func newVolumeRow(v docker.ComposeVolume) volumeRow {
	return volumeRow{
		volume: v,
		columns: []string{"  " + v.Name, "", "", "", v.Driver, "", ""},
	}
}

func (r volumeRow) Columns() []string { return r.columns }
func (r volumeRow) ID() string        { return "vol:" + r.volume.Name }

// ServicesLoadedMsg carries loaded compose resources for a project.
type ServicesLoadedMsg struct {
	Services []docker.ComposeService
	Networks []docker.ComposeNetwork
	Volumes  []docker.ComposeVolume
	Project  string
}

// ServicesModel is the Compose project resources view.
type ServicesModel struct {
	table   appui.TableModel
	filter  appui.FilterInputModel
	project string
}

// NewServicesModel creates a compose services list model.
func NewServicesModel() ServicesModel {
	columns := []appui.Column{
		{Title: "NAME"},
		{Title: "CONTAINERS", Width: 12, Fixed: true},
		{Title: "RUNNING", Width: 10, Fixed: true},
		{Title: "EXITED", Width: 10, Fixed: true},
		{Title: "IMAGE/DRIVER"},
		{Title: "HEALTH/SCOPE", Width: 14, Fixed: true},
		{Title: "PORTS"},
	}
	return ServicesModel{
		table:  appui.NewTableModel(columns),
		filter: appui.NewFilterInputModel(),
	}
}

// FilterActive returns true when the filter input is active.
func (m ServicesModel) FilterActive() bool { return m.filter.Active() }

// SetSize updates the table dimensions.
func (m *ServicesModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-2-filterH)
	m.filter.SetWidth(w)
}

// SetServices replaces the resource list with services, networks, and volumes.
func (m *ServicesModel) SetServices(services []docker.ComposeService, networks []docker.ComposeNetwork, volumes []docker.ComposeVolume, project string) {
	m.project = project
	var rows []appui.TableRow

	if len(services) > 0 {
		rows = append(rows, newSectionRow("Services", len(services)))
		for _, s := range services {
			rows = append(rows, newServiceRow(s))
		}
	}
	if len(networks) > 0 {
		rows = append(rows, newSectionRow("Networks", len(networks)))
		for _, n := range networks {
			rows = append(rows, newNetworkRow(n))
		}
	}
	if len(volumes) > 0 {
		rows = append(rows, newSectionRow("Volumes", len(volumes)))
		for _, v := range volumes {
			rows = append(rows, newVolumeRow(v))
		}
	}

	m.table.SetRows(rows)
}

// SelectedService returns the service under the cursor, or nil.
func (m ServicesModel) SelectedService() *docker.ComposeService {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if r, ok := row.(serviceRow); ok {
		return &r.service
	}
	return nil
}

// SelectedNetwork returns the network under the cursor, or nil.
func (m ServicesModel) SelectedNetwork() *docker.ComposeNetwork {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if r, ok := row.(networkRow); ok {
		return &r.network
	}
	return nil
}

// SelectedVolume returns the volume under the cursor, or nil.
func (m ServicesModel) SelectedVolume() *docker.ComposeVolume {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if r, ok := row.(volumeRow); ok {
		return &r.volume
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
	title := "Compose Resources"
	if m.project != "" {
		title = fmt.Sprintf("Compose: %s", m.project)
	}
	header := appui.RenderWidgetHeader(appui.WidgetHeaderOpts{
		Icon:     "\U0001f433",
		Title:    title,
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

// RefreshTableStyles re-applies theme styles to the inner table.
func (m *ServicesModel) RefreshTableStyles() {
	m.table.RefreshStyles()
}
