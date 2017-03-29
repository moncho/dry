package appui

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/docker/docker/api/types"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

type networksColumn struct {
	name  string // The name of the field in the struct.
	title string // Title to display in the tableHeader.
	mode  docker.SortNetworksMode
}

//DockerNetworksRenderer knows how render a container list
type DockerNetworksRenderer struct {
	columns          []networksColumn // List of columns.
	networksTemplate *template.Template
	cursor           *ui.Cursor
	daemon           docker.ContainerDaemon
	sortMode         docker.SortNetworksMode
	renderableRows   int
	startIndex       int
	endIndex         int
}

//NewDockerNetworksRenderer creates a renderer for a network list
func NewDockerNetworksRenderer(daemon docker.ContainerDaemon, cursor *ui.Cursor, sortMode docker.SortNetworksMode) *DockerNetworksRenderer {
	r := &DockerNetworksRenderer{}

	r.columns = []networksColumn{
		{`NetworkID`, `NETWORK ID`, docker.SortNetworksByID},
		{`Name`, `NAME`, docker.SortNetworksByName},
		{`Driver`, `DRIVER`, docker.SortNetworksByDriver},
		{`Containers`, `CONTAINERS`, docker.NoSortNetworks},
		{`Scope`, `SCOPE`, docker.NoSortNetworks},
	}

	r.networksTemplate = buildNetworksTemplate()
	r.cursor = cursor
	r.daemon = daemon
	r.sortMode = sortMode
	r.renderableRows = ui.ActiveScreen.Dimensions.Height - networkTableStartPos - 1
	return r
}

//SortMode sets the sort mode to use when rendering the container list
func (r *DockerNetworksRenderer) SortMode(sortMode docker.SortNetworksMode) {
	r.sortMode = sortMode
}

//Render docker ps
func (r *DockerNetworksRenderer) Render() string {
	if ok, err := r.daemon.Ok(); !ok { // If there was an error connecting to the Docker host...
		return err.Error() // then simply return the error string.
	}
	return r.networksTable()
}
func (r *DockerNetworksRenderer) networksTable() string {
	buffer := new(bytes.Buffer)
	t := tabwriter.NewWriter(buffer, 22, 0, 1, ' ', 0)
	replacer := strings.NewReplacer(`\t`, "\t", `\n`, "\n")
	fmt.Fprintln(t, replacer.Replace(r.tableHeader()))
	fmt.Fprint(t, replacer.Replace(r.networkInformation()))
	t.Flush()
	return buffer.String()
}
func (r *DockerNetworksRenderer) tableHeader() string {
	columns := make([]string, len(r.columns))
	for i, col := range r.columns {
		if r.sortMode != col.mode {
			columns[i] = col.title
		} else {
			columns[i] = DownArrow + col.title
		}
	}
	return "<green>" + strings.Join(columns, "\t") + "</>"
}

func (r *DockerNetworksRenderer) networkInformation() string {
	buf := bytes.NewBufferString("")
	networks, selected := r.networksToShow()

	context := docker.FormattingContext{
		Output:   buf,
		Template: r.networksTemplate,
		Trunc:    true,
		Selected: selected,
	}
	docker.FormatNetworks(
		context,
		networks)

	return buf.String()
}

func (r *DockerNetworksRenderer) networksToShow() ([]types.NetworkResource, int) {

	//no screen
	if r.renderableRows < 0 {
		return nil, 0
	}
	networks, _ := r.daemon.Networks()
	count := len(networks)
	cursor := ui.ActiveScreen.Cursor
	selected := cursor.Position()
	//everything fits
	if count <= r.renderableRows {
		return networks, selected
	}
	//at the the start
	if selected == 0 {
		//internal state is reset
		r.startIndex = 0
		r.endIndex = r.renderableRows
		return networks[r.startIndex : r.endIndex+1], selected
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
	visibleNetworks := networks[start:end]
	selected = findNetworkIndex(visibleNetworks, networks[selected])

	return visibleNetworks, selected
}

func buildNetworksTemplate() *template.Template {

	return template.Must(template.New(`network`).Parse(docker.DefaultNetworkTableFormat))
}

//find gets the index of the given network in the given slice
func findNetworkIndex(networks []types.NetworkResource, n types.NetworkResource) int {
	for i, network := range networks {
		if n.ID == network.ID {
			return i
		}
	}
	return -1
}
