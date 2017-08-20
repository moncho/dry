package appui

import (
	gtermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

//AskForConfirmation is an input widget to run images
type AskForConfirmation struct {
	termui.TextInput
}

//NewAskForConfirmation creates a new AskForConfirmation for the given image
func NewAskForConfirmation(question string) *AskForConfirmation {
	w := &AskForConfirmation{
		TextInput: *termui.NewTextInput(""),
	}
	w.Height = 3
	w.Width = len(question) + 4
	w.X = (ui.ActiveScreen.Dimensions.Width - w.Width) / 2
	w.Y = ui.ActiveScreen.Dimensions.Height / 2
	w.Bg = gtermui.Attribute(DryTheme.Bg)
	w.TextBgColor = gtermui.Attribute(DryTheme.Bg)
	w.TextFgColor = gtermui.ColorWhite
	w.BorderLabel = question
	w.BorderLabelFg = gtermui.ColorWhite

	return w
}

//Mount callback
func (w *AskForConfirmation) Mount() error {
	return nil
}

//Unmount callback
func (w *AskForConfirmation) Unmount() error {
	return nil
}

//Name returns the widget name
func (w *AskForConfirmation) Name() string {
	return "AskForConfirmation"
}
