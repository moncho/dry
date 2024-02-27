package swarm

import (
	"image"

	"github.com/docker/docker/api/types/swarm"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
	drytermui "github.com/moncho/dry/ui/termui"
)

// TaskRow is a Grid row showing runtime information about a task
type TaskRow struct {
	task         swarm.Task
	ID           *drytermui.ParColumn
	Name         *drytermui.ParColumn
	Image        *drytermui.ParColumn
	Node         *drytermui.ParColumn
	DesiredState *drytermui.ParColumn
	CurrentState *drytermui.ParColumn
	Error        *drytermui.ParColumn
	Ports        *drytermui.ParColumn

	drytermui.Row
}

// NewTaskRow creats a new TaskRow widget
func NewTaskRow(swarmClient docker.SwarmAPI, task swarm.Task, table drytermui.Table) *TaskRow {
	ts := formatter.NewTaskStringer(swarmClient, task, true)

	row := &TaskRow{
		task:         task,
		Name:         drytermui.NewThemedParColumn(appui.DryTheme, ts.Name()),
		Image:        drytermui.NewThemedParColumn(appui.DryTheme, ts.Image()),
		Node:         drytermui.NewThemedParColumn(appui.DryTheme, ts.NodeID()),
		DesiredState: drytermui.NewThemedParColumn(appui.DryTheme, ts.DesiredState()),
		CurrentState: drytermui.NewThemedParColumn(appui.DryTheme, ts.CurrentState()),
		Error:        drytermui.NewThemedParColumn(appui.DryTheme, ts.Error()),
		Ports:        drytermui.NewThemedParColumn(appui.DryTheme, ts.Ports()),
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.Name,
		row.Image,
		row.Node,
		row.DesiredState,
		row.CurrentState,
		row.Error,
		row.Ports,
	}
	updateStateColumns(row)

	return row

}

// Buffer returns this Row data as a termui.Buffer
func (row *TaskRow) Buffer() termui.Buffer {
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

// ColumnsForFilter returns the columns that are used to filter
func (row *TaskRow) ColumnsForFilter() []*drytermui.ParColumn {
	return []*drytermui.ParColumn{row.Name, row.Image, row.Node, row.CurrentState}
}

// Highlighted marks this rows as being highlighted
func (row *TaskRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.Fg),
		termui.Attribute(appui.DryTheme.CursorLineBg))
}

// NotHighlighted marks this rows as being not highlighted
func (row *TaskRow) NotHighlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.ListItem),
		termui.Attribute(appui.DryTheme.Bg))
}

func (row *TaskRow) changeTextColor(fg, bg termui.Attribute) {
	row.Name.TextFgColor = fg
	row.Name.TextBgColor = bg
	row.Image.TextFgColor = fg
	row.Image.TextBgColor = bg
	row.Node.TextFgColor = fg
	row.Node.TextBgColor = bg
	row.DesiredState.TextBgColor = bg
	row.CurrentState.TextBgColor = bg
	row.Error.TextFgColor = fg
	row.Error.TextBgColor = bg
	row.Ports.TextFgColor = fg
	row.Ports.TextBgColor = bg
}

// updateStateColumns changes the color of state-related column depending
// on the task state
func updateStateColumns(row *TaskRow) {
	var color termui.Attribute
	if row.DesiredState.Text == "Running" {
		color = appui.Running
	} else {
		color = appui.NotRunning
	}
	row.DesiredState.TextFgColor = color
	row.CurrentState.TextFgColor = color
	row.Error.TextFgColor = color

}
