package appui

import (
	`bytes`
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"
	`text/template`

	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/olekukonko/tablewriter"
)

type column struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
	mode  docker.SortMode
}

//DockerPs knows how render a container list
type DockerPs struct {
	layout *ui.Layout

	columns                []column // List of columns.
	containerTableTemplate *template.Template
	containerTemplate      *template.Template
	cursor                 *ui.Cursor
	daemon                 docker.ContainerDaemon
	dockerInfo             string // Docker environment information
	appHeadear             *ui.Renderer
	sortMode               docker.SortMode
}

//NewDockerPsRenderer creates a renderer for a container list
func NewDockerPsRenderer(daemon docker.ContainerDaemon, cursor *ui.Cursor, sortMode docker.SortMode) *DockerPs {
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
	r.cursor = cursor
	r.daemon = daemon
	r.sortMode = sortMode

	return r
}

//SortMode sets the sort mode to use when rendering the container list
func (r *DockerPs) SortMode(sortMode docker.SortMode) {
	r.sortMode = sortMode
}

//Render docker ps
func (r *DockerPs) Render() string {
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
func (r *DockerPs) containerTable() string {
	buffer := new(bytes.Buffer)
	t := tabwriter.NewWriter(buffer, 22, 0, 1, ' ', 0)
	replacer := strings.NewReplacer(`\t`, "\t", `\n`, "\n")
	fmt.Fprintln(t, replacer.Replace(r.tableHeader()))
	fmt.Fprint(t, replacer.Replace(r.containerInformation(r.daemon, r.cursor)))
	t.Flush()
	return buffer.String()
}
func (r *DockerPs) tableHeader() string {
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

func (r *DockerPs) containerInformation(daemon docker.ContainerDaemon, cursor *ui.Cursor) string {
	buf := bytes.NewBufferString("")
	context := docker.FormattingContext{
		Output:   buf,
		Template: r.containerTemplate,
		Trunc:    true,
		Selected: cursor.Line,
	}
	docker.Format(
		context,
		daemon.Containers())

	return buf.String()
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

func dockerInfo(daemon docker.ContainerDaemon) string {
	version, _ := daemon.Version()

	buffer := new(bytes.Buffer)

	data := [][]string{
		[]string{"Docker Host:", whiteText(daemon.DockerEnv().DockerHost), "", "Docker Version:", whiteText(version.Version)},
		[]string{"Cert Path:", whiteText(daemon.DockerEnv().DockerCertPath), "", "APIVersion:", whiteText(version.APIVersion)},
		[]string{"Verify Certificate:", whiteText(strconv.FormatBool(daemon.DockerEnv().DockerTLSVerify)),
			"",
			"OS/Arch/Kernel:",
			whiteText(version.Os + "/" + version.Arch + "/" + version.KernelVersion)},
	}

	table := tablewriter.NewWriter(buffer)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(data)
	table.Render()
	return buffer.String()
}

//-----------------------------------------------------------------------------
func arrow() string {
	return string('\U00002193')
}

//Updates the cursor position in case it is out of bounds
func updateCursorPosition(cursor *ui.Cursor, noOfContainers int) {
	if cursor.Line >= noOfContainers {
		cursor.Line = noOfContainers - 1
	} else if cursor.Line < 0 {
		cursor.Line = 0
	}
}

func whiteText(text string) string {
	return fmt.Sprintf("<yellow>%s</yellow>", text)
}
