package appui

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type jsonRenderer struct {
	data interface{}
}

// NewJSONRenderer creates a renderer that renders the given data as a JSON
func NewJSONRenderer(data interface{}) fmt.Stringer {
	return &jsonRenderer{
		data: data,
	}
}

// Render low-level information on a network
func (r *jsonRenderer) String() string {
	c, _ := json.Marshal(r.data)

	buf := new(bytes.Buffer)
	buf.WriteString("[\n")
	if err := json.Indent(buf, c, "", "    "); err == nil {
		if buf.Len() > 1 {
			// Remove trailing ','
			buf.Truncate(buf.Len() - 1)
		}
	} else {
		buf.WriteString("There was an error inspecting service information")
	}
	buf.WriteString("]\n")

	return buf.String()
}
