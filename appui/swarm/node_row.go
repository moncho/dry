package swarm

import (
	"strconv"

	"github.com/docker/docker/api/types/swarm"
	units "github.com/docker/go-units"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
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
		Name:      drytermui.NewThemedParColumn(appui.DryTheme, node.Description.Hostname),
		Role:      drytermui.NewThemedParColumn(appui.DryTheme, string(node.Spec.Role)),
		CPU:       drytermui.NewThemedParColumn(appui.DryTheme, cpus(node)),
		Memory:    drytermui.NewThemedParColumn(appui.DryTheme, units.BytesSize(float64(node.Description.Resources.MemoryBytes))),
		Engine:    drytermui.NewThemedParColumn(appui.DryTheme, node.Description.Engine.EngineVersion),
		IPAddress: drytermui.NewThemedParColumn(appui.DryTheme, node.Status.Addr),
		Status:    drytermui.NewThemedParColumn(appui.DryTheme, string(node.Status.State)),
		Height:    1,
	}
	//row.changeTextColor(termui.Attribute(appui.DryTheme.ListItem))
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
	rw := appui.CalcItemWidth(width, len(row.columns))
	for _, col := range row.columns {
		col.SetX(x)
		col.SetWidth(rw)
		x += rw + appui.DefaultColumnSpacing
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

//Highlighted marks this rows as being highlighted
func (row *NodeRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.Fg),
		termui.Attribute(appui.DryTheme.CursorLineBg))
}

//NotHighlighted marks this rows as being not highlighted
func (row *NodeRow) NotHighlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.ListItem),
		termui.Attribute(appui.DryTheme.Bg))
}

func (row *NodeRow) changeTextColor(fg, bg termui.Attribute) {
	row.Name.TextFgColor = fg
	row.Name.TextBgColor = bg
}

func cpus(node swarm.Node) string {
	//https://github.com/docker/docker/blob/v1.12.0-rc4/daemon/cluster/executor/container/container.go#L328-L332
	nano := node.Description.Resources.NanoCPUs
	nano = nano / 1e9
	return strconv.Itoa(int(nano))
}
