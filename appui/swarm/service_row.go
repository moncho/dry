package swarm

import (
	"image"
	"strings"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/api/types/swarm"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	dryformatter "github.com/moncho/dry/docker/formatter"
	drytermui "github.com/moncho/dry/ui/termui"
)

//ServiceRow is a Grid row showing service information
type ServiceRow struct {
	service      swarm.Service
	ID           *drytermui.ParColumn
	Name         *drytermui.ParColumn
	Mode         *drytermui.ParColumn
	Replicas     *drytermui.ParColumn
	Image        *drytermui.ParColumn
	ServicePorts *drytermui.ParColumn

	drytermui.Row
}

//NewServiceRow creats a new ServiceRow widget
func NewServiceRow(service swarm.Service, serviceInfo formatter.ServiceListInfo, table drytermui.Table) *ServiceRow {
	row := &ServiceRow{
		service:  service,
		ID:       drytermui.NewThemedParColumn(appui.DryTheme, service.ID),
		Name:     drytermui.NewThemedParColumn(appui.DryTheme, service.Spec.Name),
		Mode:     drytermui.NewThemedParColumn(appui.DryTheme, serviceInfo.Mode),
		Replicas: drytermui.NewThemedParColumn(appui.DryTheme, serviceInfo.Replicas),
		Image: drytermui.NewThemedParColumn(
			appui.DryTheme, serviceImage(service)),
		ServicePorts: drytermui.NewThemedParColumn(appui.DryTheme, dryformatter.FormatPorts(service.Spec.EndpointSpec.Ports)),
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.Name,
		row.Mode,
		row.Replicas,
		row.ServicePorts,
		row.Image,
	}
	return row

}

//Buffer returns this Row data as a termui.Buffer
func (row *ServiceRow) Buffer() termui.Buffer {
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

//Highlighted marks this rows as being highlighted
func (row *ServiceRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.Fg),
		termui.Attribute(appui.DryTheme.CursorLineBg))
}

//NotHighlighted marks this rows as being not highlighted
func (row *ServiceRow) NotHighlighted() {
	row.changeTextColor(
		termui.Attribute(appui.DryTheme.ListItem),
		termui.Attribute(appui.DryTheme.Bg))
}

func (row *ServiceRow) changeTextColor(fg, bg termui.Attribute) {
	row.ID.TextFgColor = fg
	row.ID.TextBgColor = bg
	row.Name.TextFgColor = fg
	row.Name.TextBgColor = bg
	row.Mode.TextFgColor = fg
	row.Mode.TextBgColor = bg
	row.Replicas.TextFgColor = fg
	row.Replicas.TextBgColor = bg
	row.ServicePorts.TextFgColor = fg
	row.ServicePorts.TextBgColor = bg
	row.Image.TextFgColor = fg
	row.Image.TextBgColor = bg
}

func serviceImage(service swarm.Service) string {
	image := service.Spec.TaskTemplate.ContainerSpec.Image
	digestMark := strings.LastIndex(image, "@")
	if digestMark > 0 {
		return image[:digestMark]
	}
	return image
}
