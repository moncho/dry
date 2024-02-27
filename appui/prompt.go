package appui

import (
	gtermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

// Prompt is a prompt widget
type Prompt struct {
	termui.TextInput
}

// NewPrompt creates a new Prompt with the given title
func NewPrompt(title string) *Prompt {
	w := &Prompt{
		TextInput: *termui.NewTextInput(ui.ActiveScreen, ""),
	}
	w.Height = 3
	w.Width = len(title) + 4
	w.X = (ui.ActiveScreen.Dimensions().Width - w.Width) / 2
	w.Y = ui.ActiveScreen.Dimensions().Height / 2
	w.Bg = gtermui.Attribute(DryTheme.Bg)
	w.TextBgColor = gtermui.Attribute(DryTheme.Bg)
	w.TextFgColor = gtermui.ColorWhite
	w.BorderLabel = title
	w.BorderLabelFg = gtermui.ColorWhite

	return w
}

// Mount callback
func (w *Prompt) Mount() error {
	return nil
}

// Unmount callback
func (w *Prompt) Unmount() error {
	return nil
}

// Name returns the widget name
func (w *Prompt) Name() string {
	return "Prompt"
}
