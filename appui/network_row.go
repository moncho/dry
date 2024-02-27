package appui

import (
	"github.com/docker/docker/api/types"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker/formatter"
	drytermui "github.com/moncho/dry/ui/termui"
)

// NetworkRow is a Grid row showing information about a Docker image
type NetworkRow struct {
	network    types.NetworkResource
	ID         *drytermui.ParColumn
	Name       *drytermui.ParColumn
	Driver     *drytermui.ParColumn
	Containers *drytermui.ParColumn
	Services   *drytermui.ParColumn
	Scope      *drytermui.ParColumn
	Subnet     *drytermui.ParColumn
	Gateway    *drytermui.ParColumn
	Row
}

// NewNetworkRow creates a new NetworkRow widget
func NewNetworkRow(network types.NetworkResource, table drytermui.Table) *NetworkRow {
	networkFormatter := formatter.NewNetworkFormatter(network, true)

	row := &NetworkRow{
		network:    network,
		ID:         drytermui.NewThemedParColumn(DryTheme, networkFormatter.ID()),
		Name:       drytermui.NewThemedParColumn(DryTheme, networkFormatter.Name()),
		Driver:     drytermui.NewThemedParColumn(DryTheme, networkFormatter.Driver()),
		Containers: drytermui.NewThemedParColumn(DryTheme, networkFormatter.Containers()),
		Services:   drytermui.NewThemedParColumn(DryTheme, networkFormatter.Services()),
		Scope:      drytermui.NewThemedParColumn(DryTheme, networkFormatter.Scope()),
		Subnet:     drytermui.NewThemedParColumn(DryTheme, networkFormatter.Subnet()),
		Gateway:    drytermui.NewThemedParColumn(DryTheme, networkFormatter.Gateway()),
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.ID,
		row.Name,
		row.Driver,
		row.Containers,
		row.Services,
		row.Scope,
		row.Subnet,
		row.Gateway,
	}
	row.ParColumns = []*drytermui.ParColumn{
		row.ID,
		row.Name,
		row.Driver,
		row.Containers,
		row.Services,
		row.Scope,
		row.Subnet,
		row.Gateway,
	}

	return row

}

// ColumnsForFilter returns the columns that are used to filter
func (row *NetworkRow) ColumnsForFilter() []*drytermui.ParColumn {
	return []*drytermui.ParColumn{row.ID, row.Name, row.Driver, row.Services, row.Scope, row.Subnet, row.Gateway}
}
