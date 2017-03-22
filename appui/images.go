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

type imagesColumn struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
	mode  docker.SortImagesMode
}

//DockerImageRenderData holds information that might be
//used during image list rendering
type DockerImageRenderData struct {
	images        []types.ImageSummary
	selectedImage int
	sortMode      docker.SortImagesMode
}

//NewDockerImageRenderData creates render data structs
func NewDockerImageRenderData(images []types.ImageSummary, selectedImage int, sortMode docker.SortImagesMode) *DockerImageRenderData {
	return &DockerImageRenderData{
		images:        images,
		selectedImage: selectedImage,
		sortMode:      sortMode,
	}
}

//DockerImagesRenderer knows how render a container list
type DockerImagesRenderer struct {
	columns             []imagesColumn // List of columns.
	imagesTableTemplate *template.Template
	imagesTemplate      *template.Template
	height              int
	data                *DockerImageRenderData
	renderLock          sync.Mutex
}

//NewDockerImagesRenderer creates a renderer for a container list
func NewDockerImagesRenderer(daemon docker.ContainerDaemon, screenHeight int) *DockerImagesRenderer {
	r := &DockerImagesRenderer{}

	r.columns = []imagesColumn{
		{`Repository`, `REPOSITORY`, docker.SortImagesByRepo},
		{`Tag`, `TAG`, docker.NoSortImages},
		{`Id`, `ID`, docker.SortImagesByID},
		{`Created`, `Created`, docker.SortImagesByCreationDate},
		{`Size`, `Size`, docker.SortImagesBySize},
	}

	r.imagesTableTemplate = buildImageTableTemplate()
	r.imagesTemplate = buildImagesTemplate()
	r.height = screenHeight
	return r
}

//PrepareForRender received information that might be used during the render phase
func (r *DockerImagesRenderer) PrepareForRender(data *DockerImageRenderData) {
	r.renderLock.Lock()
	r.data = data
	r.renderLock.Unlock()
}

//Render docker images
func (r *DockerImagesRenderer) Render() string {
	r.renderLock.Lock()
	defer r.renderLock.Unlock()

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
		if r.data.sortMode != col.mode {
			columns[i] = col.title
		} else {
			columns[i] = DownArrow + col.title
		}
	}
	return "<green>" + strings.Join(columns, "\t") + "</>"
}

func (r *DockerImagesRenderer) imageInformation() string {
	buf := bytes.NewBufferString("")
	images := r.imagesToShow()
	selected := len(images) - 1
	if r.data.selectedImage < selected {
		selected = r.data.selectedImage
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

func (r *DockerImagesRenderer) imagesToShow() []types.ImageSummary {
	images := r.data.images
	cursorPos := r.data.selectedImage
	linesForImages := r.height - imageTableStartPos - 1

	if len(images) < linesForImages {
		return images
	}

	start, end := 0, 0

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

func buildImageTableTemplate() *template.Template {
	markup := `{{.ImageTable}}`
	return template.Must(template.New(`images`).Parse(markup))
}

func buildImagesTemplate() *template.Template {

	return template.Must(template.New(`image`).Parse(docker.DefaultImageTableFormat))
}
