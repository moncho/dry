package appui

import (
	"github.com/docker/docker/api/types"
	gtermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

type ImageRunWidget struct {
	image *types.ImageSummary
	termui.TextInput
}

func NewImageRunWidget(image *types.ImageSummary) *ImageRunWidget {
	w := &ImageRunWidget{
		image:     image,
		TextInput: *termui.NewTextInput("", false),
	}
	w.Height = 3
	w.Width = ui.ActiveScreen.Dimensions.Width / 2
	w.X = (ui.ActiveScreen.Dimensions.Width - w.Width) / 2
	w.Y = ui.ActiveScreen.Dimensions.Height / 2
	w.Bg = gtermui.Attribute(DryTheme.Bg)
	w.TextBgColor = gtermui.Attribute(DryTheme.Bg)
	w.TextFgColor = gtermui.ColorWhite
	w.BorderLabel = " docker run " + image.RepoTags[0]
	w.BorderLabelFg = gtermui.ColorWhite

	return w
}

func (w *ImageRunWidget) Mount() error {
	return nil

}
func (w *ImageRunWidget) Unmount() error {

	return nil
}

func (w *ImageRunWidget) Name() string {

	return "ImageRunWidget." + w.image.ID
}
