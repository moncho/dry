package appui

import (
	"image"

	"github.com/docker/docker/api/types"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker/formatter"
	drytermui "github.com/moncho/dry/ui/termui"
)

//NetworkRow is a Grid row showing information about a Docker image
type NetworkRow struct {
	network    types.NetworkResource
	ID         *drytermui.ParColumn
	Name       *drytermui.ParColumn
	Driver     *drytermui.ParColumn
	Containers *drytermui.ParColumn
	Scope      *drytermui.ParColumn

	drytermui.Row
}

//NewNetworkRow creates a new NetworkRow widget
func NewNetworkRow(network types.NetworkResource, table drytermui.Table) *NetworkRow {
	networkFormatter := formatter.NewNetworkFormatter(network, true)

	row := &NetworkRow{
		network:    network,
		ID:         drytermui.NewThemedParColumn(DryTheme, networkFormatter.ID()),
		Name:       drytermui.NewThemedParColumn(DryTheme, networkFormatter.Name()),
		Driver:     drytermui.NewThemedParColumn(DryTheme, networkFormatter.Driver()),
		Containers: drytermui.NewThemedParColumn(DryTheme, networkFormatter.Containers()),
		Scope:      drytermui.NewThemedParColumn(DryTheme, networkFormatter.Scope()),
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.ID,
		row.Name,
		row.Driver,
		row.Containers,
		row.Scope,
	}

	return row

}

//Highlighted marks this rows as being highlighted
func (row *NetworkRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(DryTheme.Fg),
		termui.Attribute(DryTheme.CursorLineBg))
}

//NotHighlighted marks this rows as being not highlighted
func (row *NetworkRow) NotHighlighted() {

	row.changeTextColor(
		termui.Attribute(DryTheme.ListItem),
		termui.Attribute(DryTheme.Bg))
}

//Buffer returns this Row data as a termui.Buffer
func (row *NetworkRow) Buffer() termui.Buffer {
	buf := termui.NewBuffer()
	//This set the background of the whole row
	buf.Area.Min = image.Point{row.X, row.Y}
	buf.Area.Max = image.Point{row.X + row.Width, row.Y + row.Height}
	buf.Fill(' ', row.ID.TextFgColor, row.ID.TextBgColor)

	for _, col := range row.Columns {
		buf.Merge(col.Buffer())
	}
	return buf
}

func (row *NetworkRow) changeTextColor(fg, bg termui.Attribute) {

	row.ID.TextFgColor = fg
	row.ID.TextBgColor = bg
	row.Name.TextFgColor = fg
	row.Name.TextBgColor = bg
	row.Driver.TextFgColor = fg
	row.Driver.TextBgColor = bg
	row.Containers.TextFgColor = fg
	row.Containers.TextBgColor = bg
	row.Scope.TextFgColor = fg
	row.Scope.TextBgColor = bg
}
