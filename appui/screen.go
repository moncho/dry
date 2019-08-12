package appui

import (
	"image"

	termui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

// Screen is a representation of a terminal screen, limited by its
// bounds and having an active cursor.
type Screen interface {
	Bounds() image.Rectangle
	Cursor() *ui.Cursor
}

// ScreenBuffererRender is a rendering device for termui Bufferers.
type ScreenBuffererRender interface {
	Screen
	Flush() *ui.Screen
	RenderBufferer(bs ...termui.Bufferer)
}
