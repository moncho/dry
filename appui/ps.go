package appui

import (
	`bytes`
	"fmt"
	"strings"
	"sync"
	"text/tabwriter"
	`text/template`

	godocker "github.com/fsouza/go-dockerclient"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

const (
	containerTableStart = 10
)

type column struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
	mode  docker.SortMode
}

//DockerPsRenderData holds information that might be
//used during ps rendering
type DockerPsRenderData struct {
	containers        []godocker.APIContainers
	selectedContainer int
	sortMode          docker.SortMode
}

//NewDockerPsRenderData creates render data structs
func NewDockerPsRenderData(containers []godocker.APIContainers, selectedContainer int, sortMode docker.SortMode) *DockerPsRenderData {
	return &DockerPsRenderData{
		containers:        containers,
		selectedContainer: selectedContainer,
		sortMode:          sortMode,
	}
}

//DockerPs knows how render a container list
type DockerPs struct {
	appHeadear             *ui.Renderer
	columns                []column // List of columns.
	containerTableTemplate *template.Template
	containerTemplate      *template.Template
	dockerInfo             string // Docker environment information
	height                 int
	layout                 *ui.Layout
	data                   *DockerPsRenderData
	renderLock             sync.Mutex
}

//NewDockerPsRenderer creates a renderer for a container list
func NewDockerPsRenderer(daemon docker.ContainerDaemon, screenHeight int) *DockerPs {
	r := &DockerPs{}

	r.columns = []column{
		{`ID`, `CONTAINER ID`, docker.SortByContainerID},
		{`Image`, `IMAGE`, docker.SortByImage},
		{`Command`, `COMMAND`, docker.NoSort},
		{`Created`, `CREATED`, docker.NoSort},
		{`Status`, `STATUS`, docker.SortByStatus},
		{`Ports`, `PORTS`, docker.NoSort},
		{`Names`, `NAMES`, docker.SortByName},
	}
	di := dockerInfo(daemon)
	r.layout = ui.NewLayout()
	//r.layout.Header = appHeader
	r.containerTableTemplate = buildContainerTableTemplate(di)
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
	r.renderLock.Lock()
	defer r.renderLock.Unlock()

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
			columns[i] = arrow() + col.title
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

func (r *DockerPs) containersToShow() []godocker.APIContainers {
	containers := r.data.containers
	cursorPos := r.data.selectedContainer
	availableLines := r.height - containerTableStart - 1

	if len(containers) < availableLines {
		return containers
	}

	start := 0
	end := len(containers)

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

func buildContainerTableTemplate(dockerInfo string) *template.Template {
	markup := dockerInfo +
		`


{{.ContainerTable}}
`
	return template.Must(template.New(`containers`).Parse(markup))
}

func buildContainerTemplate() *template.Template {

	return template.Must(template.New(`container`).Parse(docker.DefaultTableFormat))
}

//-----------------------------------------------------------------------------
func arrow() string {
	return string('\U00002193')
}
