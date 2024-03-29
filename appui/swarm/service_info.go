package swarm

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	dryFormatter "github.com/moncho/dry/docker/formatter"
	"github.com/moncho/dry/ui"
	drytermui "github.com/moncho/dry/ui/termui"
	"github.com/olekukonko/tablewriter"
)

// ServiceInfoWidget shows service information
type ServiceInfoWidget struct {
	service     *swarm.Service
	serviceName string
	drytermui.SizableBufferer
}

// NewServiceInfoWidget creates ServiceInfoWidget with information about the service with the given ID
func NewServiceInfoWidget(swarmClient docker.SwarmAPI, service *swarm.Service, screen appui.Screen) *ServiceInfoWidget {
	name, _ := swarmClient.ResolveService(service.ID)
	info := serviceInfo(swarmClient, name, service)
	di := drytermui.NewParFromMarkupText(appui.DryTheme, info)
	di.BorderTop = false
	di.BorderBottom = true
	di.BorderLeft = false
	di.BorderRight = false
	di.BorderFg = termui.Attribute(appui.DryTheme.Footer)
	di.BorderBg = termui.Attribute(appui.DryTheme.Bg)

	di.Height = 6
	di.Width = screen.Bounds().Dx()
	di.Bg = termui.Attribute(appui.DryTheme.Bg)
	di.TextBgColor = termui.Attribute(appui.DryTheme.Bg)
	di.Display = false
	di.Y = screen.Bounds().Min.Y

	return &ServiceInfoWidget{
		serviceName:     name,
		service:         service,
		SizableBufferer: di}
}

func serviceInfo(swarmClient docker.SwarmAPI, name string, service *swarm.Service) string {

	var f ServiceListInfo
	if _, servicesInfo, err := getServiceInfo(swarmClient); err == nil {
		f = servicesInfo[service.ID]
	}

	buffer := new(bytes.Buffer)

	table := tablewriter.NewWriter(buffer)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(true)
	table.Append(
		[]string{
			ui.Blue("Service Name:"), ui.Yellow(name),
			ui.Blue("Image:"), ui.Yellow(service.Spec.TaskTemplate.ContainerSpec.Image)})
	table.Render()

	rows := [][]string{
		{
			ui.Blue("Service Mode:"), ui.Yellow(f.Mode),
			ui.Blue("Labels:"), ui.Yellow(dryFormatter.FormatLabels(service.Spec.Labels)),
			ui.Blue("Created at:"), ui.Yellow(service.CreatedAt.Format(time.RFC822)),
		},
		{
			ui.Blue("Replicas:"), ui.Yellow(f.Replicas),
			ui.Blue("Constraints:"), ui.Yellow(strings.Join(service.Spec.TaskTemplate.Placement.Constraints, ",")),
			ui.Blue("Updated at:"), ui.Yellow(service.UpdatedAt.Format(time.RFC822)),
		},
		{
			ui.Blue("Networks:"), ui.Yellow(dryFormatter.FormatSwarmNetworks(service.Spec.TaskTemplate.Networks)),
			ui.Blue("Ports:"), ui.Yellow(dryFormatter.FormatPorts(service.Spec.EndpointSpec.Ports)),
		},
		{
			ui.Blue("Configs:"), ui.Yellow(
				strconv.Itoa(
					len(service.Spec.TaskTemplate.ContainerSpec.Configs))),
			ui.Blue("Secrets:"), ui.Yellow(strconv.Itoa(
				len(service.Spec.TaskTemplate.ContainerSpec.Secrets))),
		},
	}

	table = tablewriter.NewWriter(buffer)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(true)
	table.AppendBulk(rows)
	table.Render()
	return buffer.String()
}
