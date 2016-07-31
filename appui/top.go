package appui

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/docker/engine-api/types"
	"github.com/moncho/dry/ui"
)

type topRenderer struct {
	processList *types.ContainerProcessList
}

//NewDockerTopRenderer creates renderer for docker top result
func NewDockerTopRenderer(processList *types.ContainerProcessList) ui.Renderer {
	return &topRenderer{
		processList: processList,
	}
}

func (r *topRenderer) Render() string {
	buf := bytes.NewBufferString("")

	procList := r.processList

	w := tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)

	io.WriteString(w, "<yellow><b>PROCESS LIST</></>\n\n")

	fmt.Fprintln(w,
		fmt.Sprintf("<blue>%s</>",
			strings.Join(procList.Titles, "\t")))

	for _, proc := range procList.Processes {
		fmt.Fprintln(w,
			fmt.Sprintf("<white>%s</>",
				strings.Join(proc, "\t")))
	}
	w.Flush()
	return buf.String()
}
