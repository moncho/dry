package appui

import (
	"bytes"
	"encoding/json"

	"github.com/docker/engine-api/types"
	"github.com/moncho/dry/ui"
)

type inspectRenderer struct {
	container types.ContainerJSON
}

//NewDockerInspectRenderer creates renderer for inspect information
func NewDockerInspectRenderer(container types.ContainerJSON) ui.Renderer {
	return &inspectRenderer{
		container: container,
	}
}

//Render low-level information on a container
func (r *inspectRenderer) Render() string {
	c, _ := json.Marshal(r.container)

	buf := new(bytes.Buffer)
	buf.WriteString("[\n")
	if err := json.Indent(buf, c, "", "    "); err == nil {
		if buf.Len() > 1 {
			// Remove trailing ','
			buf.Truncate(buf.Len() - 1)
		}
	} else {
		buf.WriteString("There was an error inspecting container information")
	}
	buf.WriteString("]\n")

	return buf.String()
}

type inspectImageRenderer struct {
	image types.ImageInspect
}

//NewDockerInspectImageRenderer creates renderer for image inspect information
func NewDockerInspectImageRenderer(image types.ImageInspect) ui.Renderer {
	return &inspectImageRenderer{
		image: image,
	}
}

//Render low-level information on a container
func (r *inspectImageRenderer) Render() string {
	c, _ := json.Marshal(r.image)

	buf := new(bytes.Buffer)
	buf.WriteString("[\n")
	if err := json.Indent(buf, c, "", "    "); err == nil {
		if buf.Len() > 1 {
			// Remove trailing ','
			buf.Truncate(buf.Len() - 1)
		}
	} else {
		buf.WriteString("There was an error inspecting image information")
	}
	buf.WriteString("]\n")

	return buf.String()
}

type inspectNetworkRenderer struct {
	network types.NetworkResource
}

//NewDockerInspectNetworkRenderer creates renderer for network inspect information
func NewDockerInspectNetworkRenderer(network types.NetworkResource) ui.Renderer {
	return &inspectNetworkRenderer{
		network: network,
	}
}

//Render low-level information on a network
func (r *inspectNetworkRenderer) Render() string {
	c, _ := json.Marshal(r.network)

	buf := new(bytes.Buffer)
	buf.WriteString("[\n")
	if err := json.Indent(buf, c, "", "    "); err == nil {
		if buf.Len() > 1 {
			// Remove trailing ','
			buf.Truncate(buf.Len() - 1)
		}
	} else {
		buf.WriteString("There was an error inspecting image information")
	}
	buf.WriteString("]\n")

	return buf.String()
}
