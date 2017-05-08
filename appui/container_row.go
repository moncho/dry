package appui

import (
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
	drytermui "github.com/moncho/dry/ui/termui"
)

const (
	statusSymbol = string('\u25A3')
)

//ContainerRow is a Grid row showing runtime information about a container
type ContainerRow struct {
	table     drytermui.Table
	container *docker.Container
	Indicator *drytermui.ParColumn
	ID        *drytermui.ParColumn
	Image     *drytermui.ParColumn
	Command   *drytermui.ParColumn
	Status    *drytermui.ParColumn
	Ports     *drytermui.ParColumn
	Names     *drytermui.ParColumn

	drytermui.Row
}

//NewContainerRow creats a new ContainerRow widget
func NewContainerRow(container *docker.Container, table drytermui.Table) *ContainerRow {
	cf := formatter.NewContainerFormatter(container, true)

	row := &ContainerRow{
		container: container,
		Indicator: drytermui.NewThemedParColumn(DryTheme, statusSymbol),
		ID:        drytermui.NewThemedParColumn(DryTheme, cf.ID()),
		Image:     drytermui.NewThemedParColumn(DryTheme, cf.Image()),
		Command:   drytermui.NewThemedParColumn(DryTheme, cf.Command()),
		Status:    drytermui.NewThemedParColumn(DryTheme, cf.Status()),
		Ports:     drytermui.NewThemedParColumn(DryTheme, cf.Ports()),
		Names:     drytermui.NewThemedParColumn(DryTheme, cf.Names()),
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.Indicator,
		row.ID,
		row.Image,
		row.Command,
		row.Status,
		row.Ports,
		row.Names,
	}
	if !docker.IsContainerRunning(container) {
		row.markAsNotRunning()
	} else {
		row.markAsRunning()
	}
	return row

}

//Highlighted marks this rows as being highlighted
func (row *ContainerRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(DryTheme.Fg),
		termui.Attribute(DryTheme.CursorLineBg))
}

//NotHighlighted marks this rows as being not highlighted
func (row *ContainerRow) NotHighlighted() {
	row.changeTextColor(
		termui.Attribute(DryTheme.ListItem),
		termui.Attribute(DryTheme.Bg))
}

func (row *ContainerRow) changeTextColor(fg, bg termui.Attribute) {
	row.ID.TextFgColor = fg
	row.ID.TextBgColor = bg
}

//markAsNotRunning
func (row *ContainerRow) markAsNotRunning() {
	row.Indicator.TextFgColor = NotRunning
	row.ID.TextFgColor = inactiveRowColor
	row.Image.TextFgColor = inactiveRowColor
	row.Command.TextFgColor = inactiveRowColor
	row.Status.TextFgColor = inactiveRowColor
	row.Ports.TextFgColor = inactiveRowColor
	row.Names.TextFgColor = inactiveRowColor
}

//markAsRunning
func (row *ContainerRow) markAsRunning() {
	row.Indicator.TextFgColor = Running

}
