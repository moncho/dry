package appui

import (
	"bytes"
	"strconv"

	termui "github.com/gizak/termui"
	drytermui "github.com/moncho/dry/ui/termui"

	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/olekukonko/tablewriter"
)

//DockerInfo is a bufferer for Docker info
type DockerInfo struct {
	drytermui.SizableBufferer
}

//NewDockerInfoBufferer creates a new DockerInfo
func NewDockerInfoBufferer(daemon docker.ContainerDaemon) *DockerInfo {
	di := drytermui.NewParFromMarkupText(DryTheme, dockerInfo(daemon))
	di.Border = false
	di.Height = 3
	di.Bg = termui.Attribute(DryTheme.Bg)
	di.TextBgColor = termui.Attribute(DryTheme.Bg)
	di.Display = false
	return &DockerInfo{di}
}

func dockerInfo(daemon docker.ContainerDaemon) string {
	version, _ := daemon.Version()

	buffer := new(bytes.Buffer)

	data := [][]string{
		[]string{ui.Blue("Docker Host:"), ui.Yellow(daemon.DockerEnv().DockerHost), "",
			ui.Blue("Docker Version:"), ui.Yellow(version.Version)},
		[]string{ui.Blue("Cert Path:"), ui.Yellow(daemon.DockerEnv().DockerCertPath), "",
			ui.Blue("APIVersion:"), ui.Yellow(version.APIVersion)},
		[]string{ui.Blue("Verify Certificate:"), ui.Yellow(strconv.FormatBool(daemon.DockerEnv().DockerTLSVerify)), "",
			ui.Blue("OS/Arch/Kernel:"),
			ui.Yellow(version.Os + "/" + version.Arch + "/" + version.KernelVersion)},
	}

	table := tablewriter.NewWriter(buffer)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(data)
	table.Render()
	return buffer.String()
}
