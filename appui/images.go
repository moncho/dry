package appui

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
	"text/template"

	godocker "github.com/fsouza/go-dockerclient"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

type imagesColumn struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
	mode  docker.SortImagesMode
}

//DockerImagesRenderer knows how render a container list
type DockerImagesRenderer struct {
	columns             []imagesColumn // List of columns.
	imagesTableTemplate *template.Template
	imagesTemplate      *template.Template
	cursor              *ui.Cursor
	daemon              docker.ContainerDaemon
	dockerInfo          string // Docker environment information
	sortMode            docker.SortImagesMode
	imageTableStart     int
	height              int
}

//NewDockerImagesRenderer creates a renderer for a container list
func NewDockerImagesRenderer(daemon docker.ContainerDaemon, screenHeight int, cursor *ui.Cursor, sortMode docker.SortImagesMode) *DockerImagesRenderer {
	r := &DockerImagesRenderer{}

	r.columns = []imagesColumn{
		{`Repository`, `REPOSITORY`, docker.SortImagesByRepo},
		{`Tag`, `TAG`, docker.NoSortImages},
		{`Id`, `ID`, docker.SortImagesByID},
		{`Created`, `Created`, docker.SortImagesByCreationDate},
		{`Size`, `Size`, docker.SortImagesBySize},
	}

	di := dockerInfo(daemon)

	r.imagesTableTemplate = buildImageTableTemplate(di)
	r.imagesTemplate = buildImagesTemplate()
	r.cursor = cursor
	r.daemon = daemon
	r.sortMode = sortMode
	//Safe guess about how many lines from the start of screen (including image table header) before
	//images are actually written to screen
	r.imageTableStart = 10
	r.height = screenHeight
	return r
}

//SortMode sets the sort mode to use when rendering the container list
func (r *DockerImagesRenderer) SortMode(sortMode docker.SortImagesMode) {
	r.sortMode = sortMode
}

//Render docker ps
func (r *DockerImagesRenderer) Render() string {
	if ok, err := r.daemon.Ok(); !ok { // If there was an error connecting to the Docker host...
		return err.Error() // then simply return the error string.
	}
	updateCursorPosition(r.cursor, r.daemon.ImagesCount())

	vars := struct {
		ImageTable string
	}{
		r.imagesTable(),
	}

	buffer := new(bytes.Buffer)
	r.imagesTableTemplate.Execute(buffer, vars)

	return buffer.String()
}
func (r *DockerImagesRenderer) imagesTable() string {
	buffer := new(bytes.Buffer)
	t := tabwriter.NewWriter(buffer, 22, 0, 1, ' ', 0)
	replacer := strings.NewReplacer(`\t`, "\t", `\n`, "\n")
	fmt.Fprintln(t, replacer.Replace(r.tableHeader()))
	fmt.Fprint(t, replacer.Replace(r.imageInformation()))
	t.Flush()
	return buffer.String()
}
func (r *DockerImagesRenderer) tableHeader() string {
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

func (r *DockerImagesRenderer) imageInformation() string {
	buf := bytes.NewBufferString("")
	images := r.imagesToShow()
	selected := len(images) - 1
	if r.cursor.Line < selected {
		selected = r.cursor.Line
	}
	context := docker.FormattingContext{
		Output:   buf,
		Template: r.imagesTemplate,
		Trunc:    true,
		Selected: selected,
	}
	docker.FormatImages(
		context,
		images)

	return buf.String()
}

func (r *DockerImagesRenderer) imagesToShow() []godocker.APIImages {
	images, _ := r.daemon.Images()
	cursorPos := r.cursor.Line
	linesForImages := r.height - r.imageTableStart - 1

	if len(images) < linesForImages {
		return images
	}

	start := 0
	end := len(images)

	if cursorPos > linesForImages {
		start = cursorPos + 1 - linesForImages
		end = cursorPos + 1
	} else if cursorPos == linesForImages {
		start = 1
		end = linesForImages + 1
	} else {
		start = 0
		end = linesForImages
	}

	return images[start:end]
}

func buildImageTableTemplate(dockerInfo string) *template.Template {
	markup := dockerInfo +
		`


{{.ImageTable}}
`
	return template.Must(template.New(`images`).Parse(markup))
}

func buildImagesTemplate() *template.Template {

	return template.Must(template.New(`image`).Parse(docker.DefaultImageTableFormat))
}
