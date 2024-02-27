package appui

import (
	"io"

	"github.com/gdamore/tcell"
	"github.com/moncho/dry/ui"
)

// Less renders the given renderer output in a "less" buffer
func Less(s string, screen *ui.Screen, events <-chan *tcell.EventKey, onDone func()) {
	defer onDone()
	screen.ClearAndFlush()

	less := ui.NewLess(DryTheme)
	less.MarkupSupport()
	io.WriteString(less, s)

	//Focus blocks until less decides that it does not want focus any more
	less.Focus(events)
	screen.HideCursor()
	screen.ClearAndFlush()

	screen.Sync()
}
