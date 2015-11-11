package ui

import (
	`bytes`
	"fmt"
	"strings"
	"text/tabwriter"
	`text/template`

	"github.com/moncho/dry/docker"
)

type column struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
	mode  docker.SortMode
}

type ps struct {
	columns                []column // List of columns.
	dockerInfo             string   // Docker environment information
	containerTableTemplate *template.Template
	containerTemplate      *template.Template
	daemon                 *docker.DockerDaemon
	cursor                 *Cursor
	sortMode               docker.SortMode
}

//NewDockerRenderer creates renderer for a container list
func NewDockerRenderer(daemon *docker.DockerDaemon, cursor *Cursor, sortMode docker.SortMode) Renderer {
	r := &ps{}

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
	r.containerTableTemplate = buildContainerTableTemplate(di)
	r.containerTemplate = buildContainerTemplate()
	r.cursor = cursor
	r.daemon = daemon
	r.sortMode = sortMode

	return r
}

func (r *ps) Render() string {
	if ok, err := r.daemon.Ok(); !ok { // If there was an error connecting to the Docker host...
		return err.Error() // then simply return the error string.
	}
	updateCursorPosition(r.cursor, r.daemon.ContainersCount())

	vars := struct {
		ContainerTable string
	}{
		r.containerTable(),
	}

	buffer := new(bytes.Buffer)
	r.containerTableTemplate.Execute(buffer, vars)

	return buffer.String()
}
func (r *ps) containerTable() string {
	buffer := new(bytes.Buffer)
	t := tabwriter.NewWriter(buffer, 22, 0, 1, ' ', 0)
	replacer := strings.NewReplacer(`\t`, "\t", `\n`, "\n")
	fmt.Fprintln(t, replacer.Replace(r.tableHeader()))
	fmt.Fprint(t, replacer.Replace(r.containerInformation(r.daemon, r.cursor)))
	t.Flush()
	return buffer.String()
}
func (r *ps) tableHeader() string {
	columns := make([]string, len(r.columns))
	for i, col := range r.columns {
		if r.sortMode != col.mode {
			columns[i] = col.title
		} else {
			columns[i] = arrow() + col.title
		}
	}
	return "<green>" + strings.Join(columns, "\t") + "</>"
}

func (r *ps) containerInformation(daemon *docker.DockerDaemon, cursor *Cursor) string {
	buf := bytes.NewBufferString("")
	context := docker.FormattingContext{
		Output:   buf,
		Template: r.containerTemplate,
		Trunc:    true,
		Selected: cursor.Line,
		SortMode: r.sortMode,
	}
	docker.Format(
		context,
		daemon.Containers)

	return buf.String()
}

func buildContainerTableTemplate(dockerInfo string) *template.Template {
	markup := dockerInfo + `

{{.ContainerTable}}
`
	return template.Must(template.New(`containers`).Parse(markup))
}

func buildContainerTemplate() *template.Template {

	return template.Must(template.New(`container`).Parse(docker.DefaultTableFormat))
}

func dockerInfo(daemon *docker.DockerDaemon) string {
	markup := `Docker Host: <white>{{.Env.DockerHost}}</>
{{if .Env.DockerTLSVerify}}Cert Path: <white>{{.Env.DockerCertPath}}</>
TLS:       <white>{{.Env.DockerTLSVerify}}</>
{{end}}`
	t := template.Must(template.New(`dockerinfo`).Parse(markup))
	vars := struct {
		Env *docker.DockerEnv
	}{
		daemon.DockerEnv,
	}
	buffer := new(bytes.Buffer)
	t.Execute(buffer, vars)
	return buffer.String()
}

//-----------------------------------------------------------------------------
func arrow() string {
	return string('\U00002193')
}

//Updates the cursor position in case it is out of bounds
func updateCursorPosition(cursor *Cursor, noOfContainers int) {
	if cursor.Line >= noOfContainers {
		cursor.Line = noOfContainers - 1
	} else if cursor.Line < 0 {
		cursor.Line = 0
	}
}
