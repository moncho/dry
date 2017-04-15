package appui

import (
	"strconv"

	"github.com/docker/docker/api/types/swarm"
	units "github.com/docker/go-units"
	termui "github.com/gizak/termui"
	drytermui "github.com/moncho/dry/ui/termui"
)

//NodeRow is a Grid row showing runtime information about a node
type NodeRow struct {
	node      swarm.Node
	Name      *drytermui.ParColumn
	Role      *drytermui.ParColumn
	CPU       *drytermui.ParColumn
	Memory    *drytermui.ParColumn
	Engine    *drytermui.ParColumn
	IPAddress *drytermui.ParColumn
	Status    *drytermui.ParColumn

	X, Y    int
	Width   int
	Height  int
	columns []termui.GridBufferer
}

//NewNodeRow creats a new NodeRow widget
func NewNodeRow(node swarm.Node) *NodeRow {
	row := &NodeRow{
		node:      node,
		Name:      drytermui.NewThemedParColumn(DryTheme, node.Spec.Name),
		Role:      drytermui.NewThemedParColumn(DryTheme, string(node.Spec.Role)),
		CPU:       drytermui.NewThemedParColumn(DryTheme, strconv.Itoa(int(node.Description.Resources.NanoCPUs))),
		Memory:    drytermui.NewThemedParColumn(DryTheme, units.BytesSize(float64(node.Description.Resources.MemoryBytes))),
		Engine:    drytermui.NewThemedParColumn(DryTheme, node.Description.Engine.EngineVersion),
		IPAddress: drytermui.NewThemedParColumn(DryTheme, node.Description.Hostname),
		Status:    drytermui.NewThemedParColumn(DryTheme, string(node.Status.State)),
		Height:    1,
	}
	//Columns are rendered following the slice order
	row.columns = []termui.GridBufferer{
		row.Name,
		row.Role,
		row.CPU,
		row.Memory,
		row.Engine,
		row.IPAddress,
		row.Status,
	}

	return row

}

//GetHeight returns this NodeRow heigth
func (row *NodeRow) GetHeight() int {
	return row.Height
}

//SetX sets the x position of this NodeRow
func (row *NodeRow) SetX(x int) {
	row.X = x
}

//SetY sets the y position of this NodeRow
func (row *NodeRow) SetY(y int) {
	if y == row.Y {
		return
	}
	for _, col := range row.columns {
		col.SetY(y)
	}
	row.Y = y
}

//SetWidth sets the width of this NodeRow
func (row *NodeRow) SetWidth(width int) {
	if width == row.Width {
		return
	}
	row.Width = width
	x := row.X
	rw := calcItemWidth(width, len(row.columns)-1)
	for _, col := range row.columns {
		col.SetX(x)
		col.SetWidth(rw)
		x += rw + columnSpacing
	}
}

//Buffer returns this NodeRow data as a termui.Buffer
func (row *NodeRow) Buffer() termui.Buffer {
	buf := termui.NewBuffer()
	buf.Merge(row.Name.Buffer())
	buf.Merge(row.Role.Buffer())
	buf.Merge(row.CPU.Buffer())
	buf.Merge(row.Memory.Buffer())
	buf.Merge(row.Engine.Buffer())
	buf.Merge(row.IPAddress.Buffer())
	buf.Merge(row.Status.Buffer())

	return buf
}
