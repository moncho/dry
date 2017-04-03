package appui

import (
	"bytes"
	"strconv"

	termui "github.com/gizak/termui"
	drytermui "github.com/moncho/dry/ui/termui"

	"github.com/docker/docker/api/types/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/olekukonko/tablewriter"
)

//DockerInfo is a widget to show Docker info
type DockerInfo struct {
	drytermui.SizableBufferer
}

//NewDockerInfo creates a DockerInfo widget
func NewDockerInfo(daemon docker.ContainerDaemon) *DockerInfo {
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
	info, _ := daemon.Info()

	swarmInfo := info.Swarm

	buffer := new(bytes.Buffer)

	rows := [][]string{
		[]string{
			ui.Blue("Docker Host:"), ui.Yellow(daemon.DockerEnv().DockerHost), "",
			ui.Blue("Docker Version:"), ui.Yellow(version.Version)},
		[]string{
			ui.Blue("Cert Path:"), ui.Yellow(daemon.DockerEnv().DockerCertPath), "",
			ui.Blue("APIVersion:"), ui.Yellow(version.APIVersion)},
		[]string{
			ui.Blue("Verify Certificate:"), ui.Yellow(strconv.FormatBool(daemon.DockerEnv().DockerTLSVerify)), "",
			ui.Blue("OS/Arch/Kernel:"), ui.Yellow(version.Os + "/" + version.Arch + "/" + version.KernelVersion)},
	}

	rows = addSwarmInfo(rows, swarmInfo)
	table := tablewriter.NewWriter(buffer)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(rows)
	table.Render()
	return buffer.String()
}

func addSwarmInfo(rows [][]string, info swarm.Info) [][]string {
	firstRow := rows[0]
	secondRow := rows[1]
	thirdRow := rows[2]

	firstRow = append(firstRow,
		ui.Blue("Swarm:"),
		ui.Yellow(string(info.LocalNodeState)))
	if info.LocalNodeState != swarm.LocalNodeStateInactive {

		secondRow = append(secondRow,
			ui.Blue("Cluster ID:"),
			ui.Yellow(string(info.Cluster.ID)))
		thirdRow = append(thirdRow,
			ui.Blue("Node ID:"),
			ui.Yellow(string(info.NodeID)))
	}

	return [][]string{firstRow, secondRow, thirdRow}

}
