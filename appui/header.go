package appui

import (
	"fmt"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

func WidgetHeader(what string, howMany int, info string) *termui.MarkupPar {
	par := termui.NewParFromMarkupText(DryTheme,
		fmt.Sprintf(
			"<b><blue>%s: </><yellow>%d</></>", what, howMany)+" "+info)

	par.SetX(0)
	par.Border = false
	par.Width = ui.ActiveScreen.Dimensions.Width
	par.TextBgColor = gizaktermui.Attribute(DryTheme.Bg)
	par.Bg = gizaktermui.Attribute(DryTheme.Bg)

	return par
}
