package appui

import (
	"io"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//Stream shows the content of the given stream on screen
func Stream(stream io.ReadCloser, keyboardQueue chan termbox.Event, done func()) {
	defer done()
	ui.ActiveScreen.ClearAndFlush()
	v := ui.NewLess(ui.ActiveScreen, DryTheme)
	//TODO make sure that io errors can be safely ignored
	go stdcopy.StdCopy(v, v, stream)
	v.Focus(keyboardQueue)

	stream.Close()
	termbox.HideCursor()
	ui.ActiveScreen.ClearAndFlush()
	ui.ActiveScreen.Sync()
}
