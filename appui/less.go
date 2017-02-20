package appui

import (
	"io"

	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//Less shows dry output in a "less" emulator
func Less(renderer ui.Renderer, screen *ui.Screen, keyboardQueue chan termbox.Event, closeView chan struct{}) {
	defer func() {
		closeView <- struct{}{}
	}()
	screen.Clear()
	v := ui.NewLess(DryTheme)
	v.MarkupSupport()
	io.WriteString(v, renderer.Render())

	//Focus blocks until v decides that it does not want focus any more
	if err := v.Focus(keyboardQueue); err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, closeView, err)
	}
	termbox.HideCursor()
	screen.Clear()
	screen.Sync()
}
