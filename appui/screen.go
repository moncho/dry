package appui

import (
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

// Screen is a representation of a rendering device
type Screen interface {
	Dimensions() *ui.Dimensions
	Cursor() *ui.Cursor
}

// ScreenBuffererRender is a rendering device that can render termui Bufferers
type ScreenBuffererRender interface {
	Screen
	Flush() *ui.Screen
	RenderBufferer(bs ...termui.Bufferer)
}
