package appui

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/tabwriter"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

type column struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
	mode  docker.SortMode
}

//DockerPsRenderData holds information that might be
//used during ps rendering
type DockerPsRenderData struct {
	containers []*types.Container
	sortMode   docker.SortMode
}

//NewDockerPsRenderData creates render data structs
func NewDockerPsRenderData(containers []*types.Container, sortMode docker.SortMode) *DockerPsRenderData {
	return &DockerPsRenderData{
		containers: containers,
		sortMode:   sortMode,
	}
}

//DockerPs knows how render a container list
type DockerPs struct {
	columns                  []column // List of columns.
	containerTemplate        *template.Template
	data                     *DockerPsRenderData
	renderLock               sync.RWMutex
	visibleStart, visibleEnd int
	renderableRows           int
}

//NewDockerPsRenderer creates a renderer for a container list
func NewDockerPsRenderer() *DockerPs {
	r := &DockerPs{}

	r.columns = []column{
		{`ID`, `CONTAINER`, docker.SortByContainerID},
		{`Image`, `IMAGE`, docker.SortByImage},
		{`Command`, `COMMAND`, docker.NoSort},
		{`Status`, `STATUS`, docker.SortByStatus},
		{`Ports`, `PORTS`, docker.NoSort},
		{`Names`, `NAMES`, docker.SortByName},
	}
	r.containerTemplate = buildContainerTemplate()
	r.renderableRows = ui.ActiveScreen.Dimensions.Height - containerTableStartPos - 1

	return r
}

//PrepareToRender passes information to this renderer before render time
//selected is the position, on the container list, of the selected one
func (r *DockerPs) PrepareToRender(data *DockerPsRenderData) {
	r.renderLock.Lock()
	r.data = data
	r.renderLock.Unlock()
}

//Render docker ps
func (r *DockerPs) Render() string {
	r.renderLock.RLock()
	defer r.renderLock.RUnlock()
	return r.containerTable()
}
func (r *DockerPs) containerTable() string {
	buffer := new(bytes.Buffer)
	t := tabwriter.NewWriter(buffer, 22, 0, 1, ' ', 0)
	replacer := strings.NewReplacer(`\t`, "\t", `\n`, "\n")
	fmt.Fprintln(t, replacer.Replace(r.tableHeader()))
	fmt.Fprint(t, replacer.Replace(r.containerInformation()))
	t.Flush()
	return buffer.String()
}
func (r *DockerPs) tableHeader() string {
	columns := make([]string, len(r.columns))
	for i, col := range r.columns {
		if r.data.sortMode != col.mode {
			columns[i] = col.title
		} else {
			columns[i] = DownArrow + col.title
		}
	}
	return "<green>" + strings.Join(columns, "\t") + "</>"
}

func (r *DockerPs) containerInformation() string {
	if len(r.data.containers) == 0 {
		return ""
	}
	buf := bytes.NewBufferString("")
	containers, selected := r.containersToShow()
	if selected == -1 {
		return "<red>There was an error rendering the container list, please refresh.</>"
	}

	context := docker.FormattingContext{
		Output:   buf,
		Template: r.containerTemplate,
		Trunc:    true,
		Selected: selected,
	}
	docker.Format(
		context,
		containers)

	return buf.String()
}

func (r *DockerPs) containersToShow() ([]*types.Container, int) {

	//no screen
	if r.renderableRows < 0 {
		return nil, 0
	}
	containers := r.data.containers
	count := len(containers)
	cursor := ui.ActiveScreen.Cursor
	selected := cursor.Position()
	//everything fits
	if count <= r.renderableRows {
		return containers, selected
	}
	//at the the start
	if selected == 0 {
		//internal state is reset
		r.visibleStart = 0
		r.visibleEnd = r.renderableRows
		return containers[r.visibleStart : r.visibleEnd+1], selected
	}

	if selected >= r.visibleEnd {
		if selected-r.renderableRows >= 0 {
			r.visibleStart = selected - r.renderableRows
		}
		r.visibleEnd = selected
	}
	if selected <= r.visibleStart {
		r.visibleStart = r.visibleStart - 1
		if selected+r.renderableRows < count {
			r.visibleEnd = r.visibleStart + r.renderableRows
		}
	}
	start := r.visibleStart
	end := r.visibleEnd + 1
	visibleContainers := containers[start:end]
	selected = find(visibleContainers, containers[selected])

	/*ui.ActiveScreen.Render(0, fmt.Sprintf("Page size: %d, vStart:%d, vEnd:%d, viewIndex :%d, globalIndex :%d", r.renderableRows, r.visibleStart, r.visibleEnd, selected, cursor.Position()))*/
	return visibleContainers, selected

}

func buildContainerTemplate() *template.Template {

	return template.Must(template.New(`container`).Parse(docker.DefaultTableFormat))
}

//find gets the index of the given container in the given slice
func find(containers []*types.Container, c *types.Container) int {
	for i, container := range containers {
		if c.ID == container.ID {
			return i
		}
	}
	return -1
}
