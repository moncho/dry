package appui

import (
	"github.com/moncho/dry/ui/termui"
)

type InputWidget struct {
	termui.TextInput
}

//NewInputWidget creates a widget to capture user input
func NewInputWidget(y int) *InputWidget {
	w := &InputWidget{}
	w.TextInput = *termui.NewTextInput("bla", false)
	return w

}
