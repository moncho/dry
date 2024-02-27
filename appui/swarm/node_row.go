package swarm

import (
	"image"
	"strconv"

	"github.com/docker/docker/api/types/swarm"
	units "github.com/docker/go-units"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker/formatter"
	drytermui "github.com/moncho/dry/ui/termui"
)

// NodeRow is a Grid row showing runtime information about a node
type NodeRow struct {
	node          swarm.Node
	Name          *drytermui.ParColumn
	Role          *drytermui.ParColumn
	Labels        *drytermui.ParColumn
	CPU           *drytermui.ParColumn
	Memory        *drytermui.ParColumn
	Engine        *drytermui.ParColumn
	IPAddress     *drytermui.ParColumn
	Status        *drytermui.ParColumn
	ManagerStatus *drytermui.ParColumn
	Availability  *drytermui.ParColumn

	drytermui.Row
}

// NewNodeRow creats a new NodeRow widget
func NewNodeRow(node swarm.Node, table drytermui.Table) *NodeRow {
	row := &NodeRow{
		node:          node,
		Name:          drytermui.NewThemedParColumn(appui.DryTheme, node.Description.Hostname),
		Role:          drytermui.NewThemedParColumn(appui.DryTheme, string(node.Spec.Role)),
		Labels:        drytermui.NewThemedParColumn(appui.DryTheme, formatter.FormatLabels(node.Spec.Labels)),
		CPU:           drytermui.NewThemedParColumn(appui.DryTheme, cpus(node)),
		Memory:        drytermui.NewThemedParColumn(appui.DryTheme, units.BytesSize(float64(node.Description.Resources.MemoryBytes))),
		Engine:        drytermui.NewThemedParColumn(appui.DryTheme, node.Description.Engine.EngineVersion),
		IPAddress:     drytermui.NewThemedParColumn(appui.DryTheme, node.Status.Addr),
		Status:        drytermui.NewThemedParColumn(appui.DryTheme, string(node.Status.State)),
		ManagerStatus: drytermui.NewThemedParColumn(appui.DryTheme, managerStatus(node)),
		Availability:  drytermui.NewThemedParColumn(appui.DryTheme, string(node.Spec.Availability)),
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.Name,
		row.Role,
		row.Labels,
		row.CPU,
		row.Memory,
		row.Engine,
		row.IPAddress,
		row.Status,
		row.Availability,
		row.ManagerStatus,
	}
	row.updateStatusColumn()

	return row

}

// Buffer returns this Row data as a termui.Buffer
func (row *NodeRow) Buffer() termui.Buffer {
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
func (row *NodeRow) ColumnsForFilter() []*drytermui.ParColumn {
	return []*drytermui.ParColumn{row.Name, row.Role, row.Labels, row.Status, row.Availability}
}

// Highlighted marks this rows as being highlighted
func (row *NodeRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.Fg),
		termui.Attribute(appui.DryTheme.CursorLineBg))
}

// NotHighlighted marks this rows as being not highlighted
func (row *NodeRow) NotHighlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.ListItem),
		termui.Attribute(appui.DryTheme.Bg))
}

func (row *NodeRow) changeTextColor(fg, bg termui.Attribute) {
	row.Name.TextFgColor = fg
	row.Name.TextBgColor = bg
	row.Role.TextFgColor = fg
	row.Role.TextBgColor = bg
	row.Labels.TextFgColor = fg
	row.Labels.TextBgColor = bg
	row.CPU.TextFgColor = fg
	row.CPU.TextBgColor = bg
	row.Memory.TextFgColor = fg
	row.Memory.TextBgColor = bg
	row.Engine.TextFgColor = fg
	row.Engine.TextBgColor = bg
	row.IPAddress.TextFgColor = fg
	row.IPAddress.TextBgColor = bg
	row.Engine.TextFgColor = fg
	row.Engine.TextBgColor = bg
	row.Status.TextBgColor = bg
	row.ManagerStatus.TextFgColor = fg
	row.ManagerStatus.TextBgColor = bg
	row.Availability.TextFgColor = fg
	row.Availability.TextBgColor = bg
}

func (row *NodeRow) updateStatusColumn() {
	color := appui.Running
	if row.Status.Text != "ready" {
		color = appui.NotRunning
	}
	row.Status.TextFgColor = color
}

func cpus(node swarm.Node) string {
	//https://github.com/moby/moby/blob/v1.12.0-rc4/daemon/cluster/executor/container/container.go#L328-L332
	nano := node.Description.Resources.NanoCPUs
	nano /= 1e9
	return strconv.Itoa(int(nano))
}

func managerStatus(node swarm.Node) string {
	reachability := ""
	if node.ManagerStatus != nil {
		if node.ManagerStatus.Leader {
			reachability = "Leader"
		} else {
			reachability = string(node.ManagerStatus.Reachability)
		}
	}
	return reachability
}
