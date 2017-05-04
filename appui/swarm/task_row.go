package swarm

import (
	"github.com/docker/docker/api/types/swarm"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
	drytermui "github.com/moncho/dry/ui/termui"
)

//TaskRow is a Grid row showing runtime information about a task
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

//NewTaskRow creats a new TaskRow widget
func NewTaskRow(swarmClient docker.SwarmAPI, task swarm.Task, table drytermui.Table) *TaskRow {
	ts := formatter.NewTaskStringer(swarmClient, task, true)

	row := &TaskRow{
		task:         task,
		ID:           drytermui.NewThemedParColumn(appui.DryTheme, ts.ID()),
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
		row.ID,
		row.Name,
		row.Image,
		row.Node,
		row.DesiredState,
		row.CurrentState,
		row.Error,
		row.Ports,
	}

	return row

}

//Highlighted marks this rows as being highlighted
func (row *TaskRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.Fg),
		termui.Attribute(appui.DryTheme.CursorLineBg))
}

//NotHighlighted marks this rows as being not highlighted
func (row *TaskRow) NotHighlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.ListItem),
		termui.Attribute(appui.DryTheme.Bg))
}

func (row *TaskRow) changeTextColor(fg, bg termui.Attribute) {
	row.ID.TextFgColor = fg
	row.ID.TextBgColor = bg
}
