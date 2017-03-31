package docker

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"github.com/docker/docker/api/types"
)

const (
	//DefaultTableFormat is the default table format to render a list of containers
	DefaultTableFormat = "{{.ID}}\t{{.Image}}\t{{.Command}}\t{{.Status}}\t{{.Ports}}\t{{.Names}}"
	//DefaultImageTableFormat is the default table format to render a list of images
	DefaultImageTableFormat = "{{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedSince}} ago\t{{.Size}}"
	//DefaultNetworkTableFormat is the default table format to render a list of networks
	DefaultNetworkTableFormat = "{{.ID}}\t{{.Name}}\t{{.Driver}}\t{{.Containers}}\t{{.Scope}}"
	//DefaultDiskUsageTableFormat table format to render Docker disk usage
	DefaultDiskUsageTableFormat = "{{.Type}}\t{{.Total}}\t{{.Active}}\t{{.Size}}\t{{.Reclaimable}}"
)

// FormattingContext contains information required by the formatter to print the output as desired.
type FormattingContext struct {
	// Output is the output stream to which the formatted string is written.
	Output io.Writer
	// Format is used to choose raw, table or custom format for the output.
	Template *template.Template
	// Size when set to true will display the size of the output.
	size bool
	// Quiet when set to true will simply print minimal information.
	quiet bool
	// Trunc when set to true will truncate the output of certain fields such as Container ID.
	Trunc bool
	// The selected container
	Selected int
}

// Format helps to format the output using the parameters set in the FormattingContext.
func Format(ctx FormattingContext, containers []*types.Container) {
	tableFormat(ctx, containers)
}

func tableFormat(ctx FormattingContext, containers []*types.Container) {

	var (
		buffer = bytes.NewBufferString("")
		tmpl   = ctx.Template
	)

	for index, container := range containers {
		//Sanity check
		if container != nil {
			containerCtx := &ContainerFormatter{
				trunc: ctx.Trunc,
				c:     container,
			}
			//Ugly!!
			//The lengh of both tags must be the same or the column will be displaced
			//because template execution happens before markup interpretation.
			if index == ctx.Selected {
				buffer.WriteString("<white>")
			} else {
				if IsContainerRunning(container) {
					buffer.WriteString("<cyan0>")
				} else {
					buffer.WriteString("<grey2>")
				}
			}
			if err := tmpl.Execute(buffer, containerCtx); err != nil {
				buffer = bytes.NewBufferString(fmt.Sprintf("Template parsing error: %v\n", err))
				buffer.WriteTo(ctx.Output)
				return
			}

			buffer.WriteString("</>")
			buffer.WriteString("\n")
		}
	}
	buffer.WriteTo(ctx.Output)
}

// FormatImages formats the given images.
func FormatImages(ctx FormattingContext, images []types.ImageSummary) {
	var (
		buffer = bytes.NewBufferString("")
		tmpl   = ctx.Template
	)

	for index, image := range images {
		imagerFormatter := &ImageFormatter{
			trunc: ctx.Trunc,
			image: image,
		}
		//Ugly!!
		//The lengh of both tags must be the same or the column will be displaced
		//because template execution happens before markup interpretation.
		if index == ctx.Selected {
			buffer.WriteString("<white>")
		} else {
			buffer.WriteString("<cyan0>")
		}
		if err := tmpl.Execute(buffer, imagerFormatter); err != nil {
			buffer = bytes.NewBufferString(fmt.Sprintf("Template parsing error: %v\n", err))
			buffer.WriteTo(ctx.Output)
			return
		}

		buffer.WriteString("</>")
		buffer.WriteString("\n")
	}
	buffer.WriteTo(ctx.Output)
}

// FormatNetworks formats the given slice of networks.
func FormatNetworks(ctx FormattingContext, networks []types.NetworkResource) {
	var (
		buffer = bytes.NewBufferString("")
		tmpl   = ctx.Template
	)

	for index, network := range networks {
		networkFormatter := &NetworkFormatter{
			trunc:   ctx.Trunc,
			network: network,
		}
		//Ugly!!
		//The lengh of both tags must be the same or the column will be displaced
		//because template execution happens before markup interpretation.
		if index == ctx.Selected {
			buffer.WriteString("<white>")
		} else {
			buffer.WriteString("<cyan0>")
		}
		if err := tmpl.Execute(buffer, networkFormatter); err != nil {
			buffer = bytes.NewBufferString(fmt.Sprintf("Template parsing error: %v\n", err))
			buffer.WriteTo(ctx.Output)
			return
		}

		buffer.WriteString("</>")
		buffer.WriteString("\n")
	}
	buffer.WriteTo(ctx.Output)
}
