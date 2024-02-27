package termui

import (
	gizaktermui "github.com/gizak/termui"
)

// Widget defines how a UI widget responds to its lifecycle events:
//   - Buffer returns the content of the widget as termui.Buffer,
//     it will be invoked every time the widget is render.
//   - Mount will be invoked before the Buffer method,
//     it can be used to prepare the widget for rendering.
//   - Unmount will be invoked to signal that the widget is not going
//     to be used anymore, it can be used for cleaning up.
//
// Widget are identified by its name.
type Widget interface {
	Buffer() gizaktermui.Buffer
	Mount() error
	Name() string
	Unmount() error
}
