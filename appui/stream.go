package appui

import (
	"io"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//Stream shows the content of the given stream on screen
func Stream(screen *ui.Screen, stream io.ReadCloser, keyboardQueue chan termbox.Event, closeView chan<- struct{}) {
	defer func() {
		closeView <- struct{}{}
	}()
	screen.Clear()
	screen.Sync()
	v := ui.NewLess()
	go func() {
		stdcopy.StdCopy(v, v, stream)
	}()
	if err := v.Focus(keyboardQueue); err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, closeView, err)
	}
	stream.Close()
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()
}
