package appui

import (
	"bytes"
	"strconv"

	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/olekukonko/tablewriter"
)

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
