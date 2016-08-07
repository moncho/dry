package appui

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/docker/engine-api/types"
	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
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

//NewDockerTopBufferer creates termui bufferer for docker top
func NewDockerTopBufferer(processList *types.ContainerProcessList, x, y, height, width int) (termui.Bufferer, int) {

	if processList != nil {
		buf := bytes.NewBufferString("")
		w := tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)
		lines := 3 // title + borders

		fmt.Fprintln(w,
			fmt.Sprintf("[%s](fg-blue)",
				strings.Join(processList.Titles, "\t")))

		for _, proc := range processList.Processes {
			fmt.Fprintln(w,
				fmt.Sprintf("[%s](fg-white)",
					strings.Join(proc, "\t")))
			lines++
		}
		w.Flush()
		p := termui.NewPar(buf.String())
		p.X = x
		p.Y = y
		p.Height = height
		p.Width = width
		p.TextFgColor = termui.Attribute(termbox.ColorYellow)
		p.BorderLabel = " PROCESS LIST "
		p.BorderLabelFg = termui.Attribute(termbox.ColorYellow)
		p.Border = true
		p.BorderBottom = false
		p.BorderLeft = false
		p.BorderRight = false
		p.BorderTop = true

		return p, lines
	}
	return termui.NewPar(""), 0
}
