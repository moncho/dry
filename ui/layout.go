package ui

import (
	"bytes"
	`text/template`
)

type Layout struct {
	Header   Renderer
	Content  Renderer
	Footer   Renderer
	template *template.Template
}

func NewLayout() *Layout {
	layout := &Layout{}
	layout.template = buildTemplate()
	return layout
}
func (l *Layout) Render() string {
	var header = ""
	if l.Header != nil {
		header = l.Header.Render()
	}
	var content = ""
	if l.Content != nil {
		content = l.Content.Render()
	}
	var footer = ""
	if l.Footer != nil {
		footer = l.Footer.Render()
	}
	vars := struct {
		Header  string
		Content string
		Footer  string
	}{
		header,
		content,
		footer,
	}

	buffer := new(bytes.Buffer)
	l.template.Execute(buffer, vars)
	return buffer.String()
}

func buildTemplate() *template.Template {
	markup := `{{.Header}}
{{.Content}}
{{.Footer}}
`
	return template.Must(template.New(`basicLayout`).Parse(markup))
}
