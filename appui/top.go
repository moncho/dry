package appui

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/docker/docker/api/types/container"
	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

const (
	// title + borders
	minimumHeight = 3
)

type topRenderer struct {
	processList *container.ContainerTopOKBody
}

// NewDockerTopRenderer creates renderer for docker top result
func NewDockerTopRenderer(processList *container.ContainerTopOKBody) fmt.Stringer {
	return &topRenderer{
		processList: processList,
	}
}

func (r *topRenderer) String() string {
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

// NewDockerTop creates termui bufferer for docker top
func NewDockerTop(processList *container.ContainerTopOKBody, x, y, height, width int) (termui.Bufferer, int) {

	if processList != nil {
		buf := bytes.NewBufferString("")
		w := tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)
		lines := minimumHeight // title + borders

		fmt.Fprintln(w,
			fmt.Sprintf("[%s](fg-red)",
				strings.Join(processList.Titles, "\t")))

		if len(processList.Processes) > 1 {
			pidColumnIndex := findPIDColumn(processList)
			if pidColumnIndex != -1 {
				sort.Slice(processList.Processes,
					func(i, j int) bool {
						return processList.Processes[i][pidColumnIndex] < processList.Processes[j][pidColumnIndex]
					})
			}
		}

		for _, proc := range processList.Processes {
			fmt.Fprintln(w,
				fmt.Sprintf("[%s](fg-white)",
					strings.Join(proc, "\t")))
			lines++
		}
		w.Flush()
		p := ui.NewPar(buf.String(), DryTheme)
		p.X = x
		p.Y = y
		p.Height = height - minimumHeight
		p.Width = width
		p.BorderLabel = " PROCESS LIST "
		p.Border = true
		p.BorderBottom = false
		p.BorderLeft = false
		p.BorderRight = false
		p.BorderTop = true

		if p.Height < lines {
			return p, p.Height
		}

		return p, lines
	}
	return ui.NewPar("", DryTheme), 0
}

func findPIDColumn(process *container.ContainerTopOKBody) int {
	for i, title := range process.Titles {
		if title == "PID" {
			return i
		}
	}
	return -1

}
