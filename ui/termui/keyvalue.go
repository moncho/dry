package termui

import (
	gtermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

// KeyValuePar is a widget for key value pairs
type KeyValuePar struct {
	X, Y   int
	Width  int
	Height int
	key    *gtermui.Paragraph
	value  *gtermui.Paragraph
}

// NewKeyValuePar creates a KeyValuePar widget with the given values
func NewKeyValuePar(key, value string, theme *ui.ColorTheme) *KeyValuePar {
	kv := &KeyValuePar{}
	bg := gtermui.Attribute(theme.Bg)

	kv.Height = 1
	kv.key = gtermui.NewParagraph(key + ":")
	kv.key.Border = false
	kv.key.Bg = bg
	kv.key.TextBgColor = bg
	kv.key.TextFgColor = gtermui.Attribute(theme.Key)

	kv.value = gtermui.NewParagraph(" " + value)
	kv.value.Border = false
	kv.value.Bg = bg
	kv.value.TextBgColor = bg
	kv.value.TextFgColor = gtermui.Attribute(theme.Current)

	return kv
}

// GetHeight returns this kv heigth
func (kv *KeyValuePar) GetHeight() int {
	return kv.Height
}

// SetX sets the x position of this kv
func (kv *KeyValuePar) SetX(x int) {
	kv.key.SetX(x)
	kv.value.SetX(x + 1 + len(kv.key.Text))
	kv.X = x
}

// SetY sets the y position of this kv
func (kv *KeyValuePar) SetY(y int) {
	kv.key.SetY(y)
	kv.value.SetY(y)
	kv.Y = y
}

// SetWidth sets the width of this kv
func (kv *KeyValuePar) SetWidth(width int) {
	kv.key.SetWidth(width)
	kv.value.SetWidth(width)
	kv.Width = width
}

// Buffer returns this kv data as a gtermui.Buffer
func (kv *KeyValuePar) Buffer() gtermui.Buffer {
	buf := gtermui.NewBuffer()
	buf.Merge(kv.key.Buffer())
	buf.Merge(kv.value.Buffer())
	return buf
}
