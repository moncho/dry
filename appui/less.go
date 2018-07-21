package appui

import (
	"io"

	"github.com/moncho/dry/ui"
	"github.com/nsf/termbox-go"
)

//Less renders the given renderer output in a "less" buffer
func Less(renderer ui.Renderer, screen *ui.Screen, events <-chan termbox.Event, onDone func()) {
	defer onDone()
	screen.ClearAndFlush()

	less := ui.NewLess(screen, DryTheme)
	less.MarkupSupport()
	io.WriteString(less, renderer.Render())

	//Focus blocks until less decides that it does not want focus any more
	less.Focus(events)
	termbox.HideCursor()
	screen.ClearAndFlush()

	screen.Sync()
}
