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
	screen.ClearAndFlush()
	v := ui.NewLess(screen, DryTheme)
	//TODO make sure that io errors can be safely ignored
	go stdcopy.StdCopy(v, v, stream)
	if err := v.Focus(keyboardQueue); err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, closeView, err)
	}

	stream.Close()
	termbox.HideCursor()
	screen.ClearAndFlush()
	screen.Sync()
}
