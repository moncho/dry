package appui

import (
	"github.com/docker/docker/api/types"
	gtermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

// ImageRunWidget is an input widget to run images
type ImageRunWidget struct {
	image types.ImageSummary
	termui.TextInput
}

// NewImageRunWidget creates a new ImageRunWidget for the given image
func NewImageRunWidget(image types.ImageSummary) *ImageRunWidget {
	w := &ImageRunWidget{
		image:     image,
		TextInput: *termui.NewTextInput(ui.ActiveScreen, ""),
	}
	w.Height = 3
	w.Width = ui.ActiveScreen.Dimensions().Width / 2
	w.X = (ui.ActiveScreen.Dimensions().Width - w.Width) / 2
	w.Y = ui.ActiveScreen.Dimensions().Height / 2
	w.Bg = gtermui.Attribute(DryTheme.Bg)
	w.TextBgColor = gtermui.Attribute(DryTheme.Bg)
	w.TextFgColor = gtermui.ColorWhite
	w.BorderLabel = widgetTitle(&image)
	w.BorderLabelFg = gtermui.ColorWhite

	return w
}

// Mount callback
func (w *ImageRunWidget) Mount() error {
	return nil
}

// Unmount callback
func (w *ImageRunWidget) Unmount() error {
	return nil
}

// Name returns the widget name
func (w *ImageRunWidget) Name() string {
	return "ImageRunWidget." + w.image.ID
}

func widgetTitle(image *types.ImageSummary) string {
	if len(image.RepoTags) > 0 {
		return " docker run " + image.RepoTags[0]
	} else if len(image.RepoDigests) > 0 {
		return " docker run " + image.RepoDigests[0]
	}
	return " docker run <none>"
}
