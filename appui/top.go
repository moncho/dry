package appui

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/docker/docker/api/types"
	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

const (
	// title + borders
	minimumHeight = 3
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

//NewDockerTop creates termui bufferer for docker top
func NewDockerTop(processList *types.ContainerProcessList, x, y, height, width int) (termui.Bufferer, int) {

	if processList != nil {
		buf := bytes.NewBufferString("")
		w := tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)
		lines := minimumHeight // title + borders

		fmt.Fprintln(w,
			fmt.Sprintf("[%s](fg-red)",
				strings.Join(processList.Titles, "\t")))

		//Commented because process list does not always includes
		//the same columns and sortByPid sorts by the first column
		//which is not guaranteed to be the PID.
		/*		if len(processList.Processes) > 2 {
					sort.Sort(sortByPID(processList.Processes))
				}
		*/
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

type sortByPID [][]string

func (s sortByPID) Len() int {
	return len(s)
}

func (s sortByPID) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortByPID) Less(i, j int) bool {
	return s[i][0] < s[j][0]
}
