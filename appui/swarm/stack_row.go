package swarm

import (
	"image"
	"strconv"

	termui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	drytermui "github.com/moncho/dry/ui/termui"
)

//StackRow is a Grid row showing stack information
type StackRow struct {
	stack    docker.Stack
	Name     *drytermui.ParColumn
	Services *drytermui.ParColumn

	drytermui.Row
}

//NewStackRow creats a new StackRow widget
func NewStackRow(stack docker.Stack, table drytermui.Table) *StackRow {
	row := &StackRow{
		stack:    stack,
		Name:     drytermui.NewThemedParColumn(appui.DryTheme, stack.Name),
		Services: drytermui.NewThemedParColumn(appui.DryTheme, strconv.Itoa(stack.Services)),
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.Name,
		row.Services,
	}
	return row

}

//Buffer returns this Row data as a termui.Buffer
func (row *StackRow) Buffer() termui.Buffer {
	buf := termui.NewBuffer()
	//This set the background of the whole row
	buf.Area.Min = image.Point{row.X, row.Y}
	buf.Area.Max = image.Point{row.X + row.Width, row.Y + row.Height}
	buf.Fill(' ', row.Name.TextFgColor, row.Name.TextBgColor)

	for _, col := range row.Columns {
		buf.Merge(col.Buffer())
	}
	return buf
}

//ColumnsForFilter returns the columns that are used to filter
func (row *StackRow) ColumnsForFilter() []*drytermui.ParColumn {
	return []*drytermui.ParColumn{row.Name}
}

//Highlighted marks this rows as being highlighted
func (row *StackRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.Fg),
		termui.Attribute(appui.DryTheme.CursorLineBg))
}

//NotHighlighted marks this rows as being not highlighted
func (row *StackRow) NotHighlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.ListItem),
		termui.Attribute(appui.DryTheme.Bg))
}

func (row *StackRow) changeTextColor(fg, bg termui.Attribute) {
	row.Name.TextFgColor = fg
	row.Name.TextBgColor = bg
	row.Services.TextFgColor = fg
	row.Services.TextBgColor = bg
}
