package appui

import (
	"bytes"
	"encoding/json"

	godocker "github.com/fsouza/go-dockerclient"
	"github.com/moncho/dry/ui"
)

type inspectRenderer struct {
	container *godocker.Container
}

//NewDockerInspectRenderer creates renderer for inspect information
func NewDockerInspectRenderer(container *godocker.Container) ui.Renderer {
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
