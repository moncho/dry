package appui

import (
	"fmt"
	"strings"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui/termui"
)

// SortMode sort mode for widgets
type SortMode int

// SortableColumnHeader is a column header associated to a sort mode
type SortableColumnHeader struct {
	Title string // Title to display in the tableHeader.
	Mode  SortMode
}

// WidgetHeader is a widget for the header of a widget
type WidgetHeader struct {
	elements map[string]string
	keys     []string
	Y        int
}

// NewWidgetHeader creates WidgetHeader
func NewWidgetHeader() *WidgetHeader {
	return &WidgetHeader{
		elements: make(map[string]string),
	}
}

// HeaderEntry adds a new key-value entry to this header
func (header *WidgetHeader) HeaderEntry(key, value string) {
	header.keys = append(header.keys, key)
	header.elements[key] = value
}

// GetHeight returns the widget height
func (header *WidgetHeader) GetHeight() int {
	return 1
}

// Buffer return this widget content as a termui.Buffer
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
