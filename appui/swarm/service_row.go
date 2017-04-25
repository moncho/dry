package swarm

import (
	"fmt"

	"github.com/docker/docker/api/types/swarm"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	drytermui "github.com/moncho/dry/ui/termui"
)

//ServiceRow is a Grid row showing service information
type ServiceRow struct {
	service  swarm.Service
	ID       *drytermui.ParColumn
	Name     *drytermui.ParColumn
	Mode     *drytermui.ParColumn
	Replicas *drytermui.ParColumn
	Image    *drytermui.ParColumn

	X, Y    int
	Width   int
	Height  int
	columns []termui.GridBufferer
}

//NewServiceRow creats a new ServiceRow widget
func NewServiceRow(service swarm.Service) *ServiceRow {
	row := &ServiceRow{
		service: service,
		ID:      drytermui.NewThemedParColumn(appui.DryTheme, service.ID),
		Name:    drytermui.NewThemedParColumn(appui.DryTheme, service.Spec.Name),
		Mode:    drytermui.NewThemedParColumn(appui.DryTheme, serviceMode(service)),
		Replicas: drytermui.NewThemedParColumn(appui.DryTheme,
			fmt.Sprintf("%d/%d", 0, *service.Spec.Mode.Replicated.Replicas)),
		Image:  drytermui.NewThemedParColumn(appui.DryTheme, service.Spec.TaskTemplate.ContainerSpec.Image),
		Height: 1,
	}
	//Columns are rendered following the slice order
	row.columns = []termui.GridBufferer{
		row.ID,
		row.Name,
		row.Mode,
		row.Replicas,
		row.Image,
	}
	return row

}

//GetHeight returns this ServiceRow heigth
func (row *ServiceRow) GetHeight() int {
	return row.Height
}

//SetX sets the x position of this ServiceRow
func (row *ServiceRow) SetX(x int) {
	row.X = x
}

//SetY sets the y position of this ServiceRow
func (row *ServiceRow) SetY(y int) {
	if y == row.Y {
		return
	}
	for _, col := range row.columns {
		col.SetY(y)
	}
	row.Y = y
}

//SetWidth sets the width of this ServiceRow
func (row *ServiceRow) SetWidth(width int) {
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

//Buffer returns this ServiceRow data as a termui.Buffer
func (row *ServiceRow) Buffer() termui.Buffer {

	buf := termui.NewBuffer()

	buf.Merge(row.ID.Buffer())
	buf.Merge(row.Name.Buffer())
	buf.Merge(row.Mode.Buffer())
	buf.Merge(row.Replicas.Buffer())
	buf.Merge(row.Image.Buffer())

	return buf
}

//Highlighted marks this rows as being highlighted
func (row *ServiceRow) Highlighted() {
	row.changeTextColor(termui.Attribute(appui.DryTheme.CursorLineBg))
}

//NotHighlighted marks this rows as being not highlighted
func (row *ServiceRow) NotHighlighted() {
	row.changeTextColor(termui.Attribute(appui.DryTheme.Bg))
}

func (row *ServiceRow) changeTextColor(color termui.Attribute) {
	row.ID.TextBgColor = color
}

func serviceMode(service swarm.Service) string {
	if service.Spec.Mode.Global != nil {
		return "global"
	}
	return "replicated"
}
