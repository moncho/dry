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
	"github.com/moncho/dry/docker/formatter"
	"github.com/moncho/dry/ui"
)

type imagesColumn struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
	mode  docker.SortImagesMode
}

//DockerImageRenderData holds information that might be
//used during image list rendering
type DockerImageRenderData struct {
	images   []types.ImageSummary
	sortMode docker.SortImagesMode
}

//NewDockerImageRenderData creates render data structs
func NewDockerImageRenderData(images []types.ImageSummary, sortMode docker.SortImagesMode) *DockerImageRenderData {
	return &DockerImageRenderData{
		images:   images,
		sortMode: sortMode,
	}
}

//DockerImagesRenderer knows how render a container list
type DockerImagesRenderer struct {
	columns        []imagesColumn // List of columns.
	imagesTemplate *template.Template
	data           *DockerImageRenderData
	renderLock     sync.Mutex
	renderableRows int
	startIndex     int
	endIndex       int
}

//NewDockerImagesRenderer creates a renderer for a container list
func NewDockerImagesRenderer() *DockerImagesRenderer {
	r := &DockerImagesRenderer{}

	r.columns = []imagesColumn{
		{`Repository`, `REPOSITORY`, docker.SortImagesByRepo},
		{`Tag`, `TAG`, docker.NoSortImages},
		{`Id`, `ID`, docker.SortImagesByID},
		{`Created`, `Created`, docker.SortImagesByCreationDate},
		{`Size`, `Size`, docker.SortImagesBySize},
	}

	r.imagesTemplate = buildImagesTemplate()
	r.renderableRows = ui.ActiveScreen.Dimensions.Height - networkTableStartPos - 1

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
	return r.imagesTable()

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
	if len(r.data.images) == 0 {
		return ""
	}
	buf := bytes.NewBufferString("")
	images, selected := r.imagesToShow()

	context := formatter.FormattingContext{
		Output:   buf,
		Template: r.imagesTemplate,
		Trunc:    true,
		Selected: selected,
	}
	formatter.FormatImages(
		context,
		images)

	return buf.String()
}

func (r *DockerImagesRenderer) imagesToShow() ([]types.ImageSummary, int) {

	//no screen
	if r.renderableRows < 0 {
		return nil, 0
	}
	images := r.data.images
	count := len(images)
	cursor := ui.ActiveScreen.Cursor
	selected := cursor.Position()
	//everything fits
	if count <= r.renderableRows {
		return images, selected
	}
	//at the the start
	if selected == 0 {
		//internal state is reset
		r.startIndex = 0
		r.endIndex = r.renderableRows
		return images[r.startIndex : r.endIndex+1], selected
	}

	if selected >= r.endIndex {
		if selected-r.renderableRows >= 0 {
			r.startIndex = selected - r.renderableRows
		}
		r.endIndex = selected
	}
	if selected <= r.startIndex {
		r.startIndex = r.startIndex - 1
		if selected+r.renderableRows < count {
			r.endIndex = r.startIndex + r.renderableRows
		}
	}
	start := r.startIndex
	end := r.endIndex + 1
	visibleImages := images[start:end]
	selected = findImageIndex(visibleImages, images[selected])

	return visibleImages, selected
}

func buildImagesTemplate() *template.Template {

	return template.Must(template.New(`image`).Parse(formatter.DefaultImageTableFormat))
}

//find gets the index of the given network in the given slice
func findImageIndex(networks []types.ImageSummary, n types.ImageSummary) int {
	for i, network := range networks {
		if n.ID == network.ID {
			return i
		}
	}
	return -1
}
