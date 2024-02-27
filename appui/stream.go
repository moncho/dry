package appui

import (
	"io"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gdamore/tcell"
	"github.com/moncho/dry/ui"
)

// Stream shows the content of the given stream on screen
func Stream(stream io.ReadCloser, keyboardQueue <-chan *tcell.EventKey, done func()) {
	defer done()
	ui.ActiveScreen.ClearAndFlush()
	v := ui.NewLess(DryTheme)
	//TODO do something with io errors
	go stdcopy.StdCopy(v, v, stream)
	v.Focus(keyboardQueue)

	stream.Close()
	ui.ActiveScreen.HideCursor()
	ui.ActiveScreen.ClearAndFlush()
	ui.ActiveScreen.Sync()
}
