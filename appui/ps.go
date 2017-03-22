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
)

type column struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
	mode  docker.SortMode
}

//DockerPsRenderData holds information that might be
//used during ps rendering
type DockerPsRenderData struct {
	containers        []*types.Container
	selectedContainer int
	sortMode          docker.SortMode
}

//NewDockerPsRenderData creates render data structs
func NewDockerPsRenderData(containers []*types.Container, selectedContainer int, sortMode docker.SortMode) *DockerPsRenderData {
	return &DockerPsRenderData{
		containers:        containers,
		selectedContainer: selectedContainer,
		sortMode:          sortMode,
	}
}

//DockerPs knows how render a container list
type DockerPs struct {
	columns                []column // List of columns.
	containerTableTemplate *template.Template
	containerTemplate      *template.Template
	dockerInfo             string // Docker environment information
	height                 int
	data                   *DockerPsRenderData
	renderLock             sync.RWMutex
}

//NewDockerPsRenderer creates a renderer for a container list
func NewDockerPsRenderer(screenHeight int) *DockerPs {
	r := &DockerPs{}

	r.columns = []column{
		{`ID`, `CONTAINER`, docker.SortByContainerID},
		{`Image`, `IMAGE`, docker.SortByImage},
		{`Command`, `COMMAND`, docker.NoSort},
		{`Status`, `STATUS`, docker.SortByStatus},
		{`Ports`, `PORTS`, docker.NoSort},
		{`Names`, `NAMES`, docker.SortByName},
	}
	r.containerTableTemplate = buildContainerTableTemplate()
	r.containerTemplate = buildContainerTemplate()
	r.height = screenHeight

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

	vars := struct {
		ContainerTable string
	}{
		r.containerTable(),
	}

	buffer := new(bytes.Buffer)
	r.containerTableTemplate.Execute(buffer, vars)

	return buffer.String()
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
	buf := bytes.NewBufferString("")
	containers := r.containersToShow()

	//From the containers-to-show list
	//decide which one is selected
	selected := len(containers) - 1
	if r.data.selectedContainer < selected {
		selected = r.data.selectedContainer
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

func (r *DockerPs) containersToShow() []*types.Container {
	containers := r.data.containers
	cursorPos := r.data.selectedContainer
	availableLines := r.height - containerTableStartPos - 1

	if len(containers) < availableLines {
		return containers
	}

	start, end := 0, 0

	if cursorPos > availableLines {
		start = cursorPos + 1 - availableLines
		end = cursorPos + 1
	} else if cursorPos == availableLines {
		start = 1
		end = availableLines + 1
	} else {
		start = 0
		end = availableLines
	}

	return containers[start:end]
}

func buildContainerTableTemplate() *template.Template {
	markup := `{{.ContainerTable}}`
	return template.Must(template.New(`containers`).Parse(markup))
}

func buildContainerTemplate() *template.Template {

	return template.Must(template.New(`container`).Parse(docker.DefaultTableFormat))
}
