package appui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/docker/docker/api/types/volume"
	"github.com/moncho/dry/docker"
)

// volumeRow wraps a Docker volume as a TableRow.
type volumeRow struct {
	volume  *volume.Volume
	columns []string
}

func newVolumeRow(v *volume.Volume) volumeRow {
	return volumeRow{
		volume: v,
		columns: []string{
			v.Driver, v.Name, v.Mountpoint,
		},
	}
}

func (r volumeRow) Columns() []string { return r.columns }
func (r volumeRow) ID() string        { return r.volume.Name }

// VolumesLoadedMsg carries the loaded volumes.
type VolumesLoadedMsg struct {
	Volumes []*volume.Volume
}

// VolumesModel is the volumes list view sub-model.
type VolumesModel struct {
	table  TableModel
	filter FilterInputModel
	daemon docker.ContainerDaemon
}

// NewVolumesModel creates a volumes list model.
func NewVolumesModel() VolumesModel {
	columns := []Column{
		{Title: "DRIVER", Width: 16, Fixed: true},
		{Title: "NAME"},
		{Title: "MOUNTPOINT"},
	}
	return VolumesModel{
		table:  NewTableModel(columns),
		filter: NewFilterInputModel(),
	}
}

// SetDaemon sets the Docker daemon reference.
func (m *VolumesModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// SetSize updates the table dimensions.
func (m *VolumesModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-1-filterH)
	m.filter.SetWidth(w)
}

// SetVolumes replaces the volume list.
func (m *VolumesModel) SetVolumes(volumes []*volume.Volume) {
	rows := make([]TableRow, len(volumes))
	for i, v := range volumes {
		rows[i] = newVolumeRow(v)
	}
	m.table.SetRows(rows)
}

// SelectedVolume returns the volume under the cursor, or nil.
func (m VolumesModel) SelectedVolume() *volume.Volume {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if vr, ok := row.(volumeRow); ok {
		return vr.volume
	}
	return nil
}

// Update handles volume-list-specific key events.
func (m VolumesModel) Update(msg tea.Msg) (VolumesModel, tea.Cmd) {
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

// View renders the volumes list.
func (m VolumesModel) View() string {
	header := m.widgetHeader()
	tableView := m.table.View()
	result := header + "\n" + tableView
	if filterView := m.filter.View(); filterView != "" {
		result += "\n" + filterView
	}
	return result
}

func (m VolumesModel) widgetHeader() string {
	return RenderWidgetHeader(WidgetHeaderOpts{
		Icon:     "💾",
		Title:    "Volumes",
		Total:    m.table.TotalRowCount(),
		Filtered: m.table.RowCount(),
		Filter:   m.table.FilterText(),
		Width:    m.table.Width(),
		Accent:   DryTheme.Warning,
	})
}
