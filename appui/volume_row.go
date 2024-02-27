package appui

import (
	"github.com/docker/docker/api/types/volume"
	termui "github.com/gizak/termui"
	drytermui "github.com/moncho/dry/ui/termui"
)

// VolumeRow is a Grid row showing information about a Docker volume.
type VolumeRow struct {
	volume *volume.Volume
	Driver *drytermui.ParColumn
	Name   *drytermui.ParColumn
	Row
}

// NewVolumeRow creates VolumeRow widgets.
func NewVolumeRow(volume *volume.Volume, table drytermui.Table) *VolumeRow {

	row := &VolumeRow{
		volume: volume,
		Driver: drytermui.NewThemedParColumn(DryTheme, volume.Driver),
		Name:   drytermui.NewThemedParColumn(DryTheme, volume.Name),
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.Driver,
		row.Name,
	}
	row.ParColumns = []*drytermui.ParColumn{
		row.Driver,
		row.Name,
	}

	return row

}

// ColumnsForFilter returns the columns that are used to filter
func (row *VolumeRow) ColumnsForFilter() []*drytermui.ParColumn {
	return []*drytermui.ParColumn{row.Name, row.Driver}
}
