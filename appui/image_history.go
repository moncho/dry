package appui

import (
	"bytes"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/go-units"
	drydocker "github.com/moncho/dry/docker"

	"github.com/moncho/dry/ui"
	"github.com/olekukonko/tablewriter"
)

//DockerImageHistoryRenderer knows how render history image
type DockerImageHistoryRenderer struct {
	imageHistory []types.ImageHistory
}

//NewDockerImageHistoryRenderer creates a renderer for the history of an image
func NewDockerImageHistoryRenderer(imageHistory []types.ImageHistory) ui.Renderer {
	r := &DockerImageHistoryRenderer{imageHistory: imageHistory}

	return r
}

//Render docker ps
func (r *DockerImageHistoryRenderer) Render() string {

	buffer := new(bytes.Buffer)

	table := tablewriter.NewWriter(buffer)
	table.SetHeader([]string{"IMAGE", "CREATED", "CREATED BY", "SIZE", "COMMENT"})
	table.SetBorder(false)
	table.SetColumnSeparator(" ")
	//table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(true)

	for _, history := range r.imageHistory {
		table.Append(historyColumns(history))
	}
	table.Render()
	return ui.White(buffer.String())
}

func historyColumns(history types.ImageHistory) []string {
	result := make([]string, 5)

	if strings.HasPrefix(history.ID, "<") {
		result[0] = history.ID
	} else {
		result[0] = (drydocker.ShortImageID(history.ID))
	}
	result[1] = drydocker.DurationForHumans(history.Created)
	result[2] = history.CreatedBy
	result[3] = units.HumanSize(float64(history.Size))
	if history.Tags != nil {
		result[4] = strings.Join(history.Tags, ", ")
	}
	return result
}
