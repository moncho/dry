package appui

import (
	"fmt"
	"github.com/moncho/dry/docker"
	"strings"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui/termui"
)

type sortMode = docker.SortMode

//SortableColumnHeader is a column header associated to a  sort mode
type SortableColumnHeader struct {
	Title string // Title to display in the tableHeader.
	Mode  sortMode
}

type WidgetHeader struct {
	elements map[string]string
	keys     []string
	Y        int
}

func NewWidgetHeader() *WidgetHeader {
	return &WidgetHeader{
		elements: make(map[string]string),
	}
}
func (header *WidgetHeader) HeaderEntry(key, value string) {
	header.keys = append(header.keys, key)
	header.elements[key] = value
}
func (header *WidgetHeader) GetHeight() int {
	return 1
}

// Buffer return this paragraph content as a termui.Buffer
func (header *WidgetHeader) Buffer() gizaktermui.Buffer {
	var entries []string
	width := 0
	for _, k := range header.keys {
		entries = append(entries,
			fmt.Sprintf("<b><blue>%s: </><yellow>%s</></>", k, header.elements[k]))
		width += len(k) + len(header.elements[k])
	}
	s := strings.Join(entries, " <blue>|</> ")
	par := termui.NewParFromMarkupText(DryTheme, s)

	par.SetX(0)
	par.SetY(header.Y)
	par.Border = false
	par.Width = len([]rune(s))
	par.TextBgColor = gizaktermui.Attribute(DryTheme.Bg)
	par.Bg = gizaktermui.Attribute(DryTheme.Bg)
	return par.Buffer()
}
