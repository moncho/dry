package docker

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"github.com/fsouza/go-dockerclient"
)

const (
	//DefaultTableFormat is the default table format to render a list of containers
	DefaultTableFormat = "{{.ID}}\t{{.Image}}\t{{.Command}}\t{{.RunningFor}} ago\t{{.Status}}\t{{.Ports}}\t{{.Names}}"
	//DefaultQuietFormat = "{{.ID}}"
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
func Format(ctx FormattingContext, containers []docker.APIContainers) {
	tableFormat(ctx, containers)
}

func tableFormat(ctx FormattingContext, containers []docker.APIContainers) {

	var (
		buffer = bytes.NewBufferString("")
		tmpl   = ctx.Template
	)

	for index, container := range containers {
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
	buffer.WriteTo(ctx.Output)
}
