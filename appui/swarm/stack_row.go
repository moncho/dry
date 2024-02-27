package swarm

import (
	"strconv"

	termui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	drytermui "github.com/moncho/dry/ui/termui"
)

// StackRow is a Grid row showing stack information
type StackRow struct {
	stack        docker.Stack
	Name         *drytermui.ParColumn
	Services     *drytermui.ParColumn
	Orchestrator *drytermui.ParColumn
	Networks     *drytermui.ParColumn
	Configs      *drytermui.ParColumn
	Secrets      *drytermui.ParColumn

	appui.Row
}

// NewStackRow creats a new StackRow widget
func NewStackRow(stack docker.Stack, table drytermui.Table) *StackRow {
	row := &StackRow{
		stack:        stack,
		Name:         drytermui.NewThemedParColumn(appui.DryTheme, stack.Name),
		Services:     drytermui.NewThemedParColumn(appui.DryTheme, strconv.Itoa(stack.Services)),
		Orchestrator: drytermui.NewThemedParColumn(appui.DryTheme, stack.Orchestrator),
		Networks:     drytermui.NewThemedParColumn(appui.DryTheme, strconv.Itoa(stack.Networks)),
		Configs:      drytermui.NewThemedParColumn(appui.DryTheme, strconv.Itoa(stack.Configs)),
		Secrets:      drytermui.NewThemedParColumn(appui.DryTheme, strconv.Itoa(stack.Secrets)),
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.Name,
		row.Services,
		row.Orchestrator,
		row.Networks,
		row.Configs,
		row.Secrets,
	}
	row.ParColumns = []*drytermui.ParColumn{
		row.Name,
		row.Services,
		row.Orchestrator,
		row.Networks,
		row.Configs,
		row.Secrets,
	}
	return row

}

// ColumnsForFilter returns the columns that are used to filter
func (row *StackRow) ColumnsForFilter() []*drytermui.ParColumn {
	return []*drytermui.ParColumn{row.Name, row.Orchestrator}
}
