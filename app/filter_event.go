package app

import (
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
)

func showFilterInput(es ui.EventSource, onDone func(string, bool)) {
	rw := appui.NewPrompt("Filter? (blank to remove current filter)")
	widgets.add(rw)
	go func() {
		rw.OnFocus(es)
		widgets.remove(rw)
		onDone(rw.Text())
	}()
}
