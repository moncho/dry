package swarm

import (
	"strings"

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

	appui.Row
}

//NewServiceRow creats a new ServiceRow widget
func NewServiceRow(service swarm.Service, serviceInfo ServiceListInfo, table drytermui.Table) *ServiceRow {
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
	row.ParColumns = []*drytermui.ParColumn{
		row.Name,
		row.Mode,
		row.Replicas,
		row.ServicePorts,
		row.Image,
	}
	return row

}

//ColumnsForFilter returns the columns that are used to filter
func (row *ServiceRow) ColumnsForFilter() []*drytermui.ParColumn {
	return []*drytermui.ParColumn{row.Name, row.Image, row.Mode}
}

func serviceImage(service swarm.Service) string {
	image := service.Spec.TaskTemplate.ContainerSpec.Image
	digestMark := strings.LastIndex(image, "@")
	if digestMark > 0 {
		return image[:digestMark]
	}
	return image
}
