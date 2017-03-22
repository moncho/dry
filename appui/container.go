package appui

import (
	"bytes"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/olekukonko/tablewriter"
)

const (
	maxWidth = 80
)

//NewContainerInfo returns detailed container information. Returned int value
//is the number of lines.
func NewContainerInfo(container *types.Container) (string, int) {

	buffer := new(bytes.Buffer)
	var status string
	if docker.IsContainerRunning(container) {
		status = ui.Yellow(container.Status)
	} else {
		status = ui.Red(container.Status)
	}
	lines := len(container.Command) / maxWidth
	data := [][]string{
		[]string{ui.Blue("Container Name:"), ui.Yellow(container.Names[0]), ui.Blue("ID:"), ui.Yellow(docker.TruncateID(container.ID)), ui.Blue("Status:"), status},
		[]string{ui.Blue("Image:"), ui.Yellow(container.Image), ui.Blue("Created:"), ui.Yellow(docker.DurationForHumans(container.Created) + " ago")},
		[]string{ui.Blue("Command:"), ui.Yellow(container.Command)},
		[]string{ui.Blue("Port mapping:"), ui.Yellow(docker.DisplayablePorts(container.Ports))},
	}
	var networkNames []string
	var networkIps []string
	for k, v := range container.NetworkSettings.Networks {
		networkNames = append(networkNames, ui.Blue("Network Name: "))
		networkNames = append(networkNames, ui.Yellow(k))
		networkIps = append(networkIps, ui.Blue("\tIP Address:"))
		networkIps = append(networkIps, ui.Yellow(v.IPAddress))
	}
	data = append(data, networkNames)
	data = append(data, networkIps)

	data = append(data, []string{ui.Blue("Labels"), ui.Yellow(
		strconv.Itoa(len(container.Labels)))})

	table := tablewriter.NewWriter(buffer)
	table.SetAutoFormatHeaders(false)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColWidth(maxWidth)
	table.AppendBulk(data)
	table.Render()
	return buffer.String(), len(data) + lines
}
