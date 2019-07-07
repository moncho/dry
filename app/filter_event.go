package app

import (
	"fmt"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
)

func showFilterInput(es ui.EventSource, onDone func(string, bool)) {
	rw := appui.NewPrompt("Filter? (blank to remove current filter)")
	widgets.add(rw)
	go func() {
		err := rw.OnFocus(es)
		if err != nil {
			fmt.Println(err)
		}
		widgets.remove(rw)
		onDone(rw.Text())
	}()
}
