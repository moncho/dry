package appui

import (
	"io"

	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//Less renders the given renderer output in a "less" buffer
func Less(renderer ui.Renderer, screen *ui.Screen, keyboardQueue chan termbox.Event, closeView chan struct{}) {
	defer func() {
		closeView <- struct{}{}
	}()
	screen.ClearAndFlush()

	less := ui.NewLess(screen, DryTheme)
	less.MarkupSupport()
	io.WriteString(less, renderer.Render())

	//Focus blocks until less decides that it does not want focus any more
	if err := less.Focus(keyboardQueue); err != nil {
		ui.ShowErrorMessage(screen, keyboardQueue, closeView, err)
	}
	termbox.HideCursor()
	screen.ClearAndFlush()

	screen.Sync()
}
