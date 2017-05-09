package swarm

import (
	"strconv"

	"github.com/docker/docker/api/types/swarm"
	units "github.com/docker/go-units"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker/formatter"
	drytermui "github.com/moncho/dry/ui/termui"
)

//NodeRow is a Grid row showing runtime information about a node
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

	drytermui.Row
}

//NewNodeRow creats a new NodeRow widget
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
		row.ManagerStatus,
	}
	row.updateStatusColumn()

	return row

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

func (row *NodeRow) updateStatusColumn() {
	color := appui.Running
	if row.Status.Text != "ready" {
		color = appui.NotRunning
	}
	row.Status.TextFgColor = color
}

func cpus(node swarm.Node) string {
	//https://github.com/docker/docker/blob/v1.12.0-rc4/daemon/cluster/executor/container/container.go#L328-L332
	nano := node.Description.Resources.NanoCPUs
	nano = nano / 1e9
	return strconv.Itoa(int(nano))
}

func managerStatus(node swarm.Node) string {
	if node.ManagerStatus != nil && node.ManagerStatus.Leader {
		return "Leader"
	}
	return ""
}
