package swarm

import (
	"github.com/docker/docker/api/types/swarm"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
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

	X, Y    int
	Width   int
	Height  int
	columns []termui.GridBufferer
}

//NewTaskRow creats a new TaskRow widget
func NewTaskRow(task swarm.Task) *TaskRow {
	ts := formatter.NewTaskStringer(task)
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
		Height:       1,
	}
	row.changeTextColor(termui.Attribute(appui.DryTheme.ListItem))
	//Columns are rendered following the slice order
	row.columns = []termui.GridBufferer{
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

//GetHeight returns this TaskRow heigth
func (row *TaskRow) GetHeight() int {
	return row.Height
}

//SetX sets the x position of this TaskRow
func (row *TaskRow) SetX(x int) {
	row.X = x
}

//SetY sets the y position of this TaskRow
func (row *TaskRow) SetY(y int) {
	if y == row.Y {
		return
	}
	for _, col := range row.columns {
		col.SetY(y)
	}
	row.Y = y
}

//SetWidth sets the width of this TaskRow
func (row *TaskRow) SetWidth(width int) {
	if width == row.Width {
		return
	}
	row.Width = width
	x := row.X
	rw := appui.CalcItemWidth(width, len(row.columns))
	for _, col := range row.columns {
		col.SetX(x)
		col.SetWidth(rw)
		x += rw + appui.DefaultColumnSpacing
	}
}

//Buffer returns this TaskRow data as a termui.Buffer
func (row *TaskRow) Buffer() termui.Buffer {

	buf := termui.NewBuffer()
	buf.Merge(row.ID.Buffer())
	buf.Merge(row.Image.Buffer())
	buf.Merge(row.Node.Buffer())
	buf.Merge(row.DesiredState.Buffer())
	buf.Merge(row.CurrentState.Buffer())
	buf.Merge(row.Error.Buffer())
	buf.Merge(row.Ports.Buffer())

	return buf
}

//Highlighted marks this rows as being highlighted
func (row *TaskRow) Highlighted() {
	row.changeTextColor(termui.Attribute(appui.DryTheme.SelectedListItem))
}

//NotHighlighted marks this rows as being not highlighted
func (row *TaskRow) NotHighlighted() {
	row.changeTextColor(termui.Attribute(appui.DryTheme.ListItem))
}

func (row *TaskRow) changeTextColor(color termui.Attribute) {
	row.Name.TextFgColor = color
	row.ID.TextFgColor = color
	row.Image.TextFgColor = color
	row.Node.TextFgColor = color
	row.DesiredState.TextFgColor = color
	row.CurrentState.TextFgColor = color
	row.Error.TextFgColor = color
	row.Ports.TextFgColor = color

}
