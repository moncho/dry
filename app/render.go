package app

import (
	"fmt"
	"io"
	"time"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
)

type viewMode uint16

//known view modes
const (
	Main viewMode = iota
	HelpMode
	StatsMode
	InspectMode
	InfoMode
)
const (
	menuBarBackgroundColor = 0x19
)

//Render renders dry in the given screen
func Render(d *Dry, screen *ui.Screen, status *ui.StatusBar) {
	switch d.State.viewMode {
	case Main:
		{
			//after a refresh, sorting is needed
			d.dockerDaemon.Sort(d.State.SortMode)
			d.renderer.SortMode(d.State.SortMode)
			status.Render()
			screen.RenderLine(0, 0, `<right><white>`+time.Now().Format(`15:04:05`)+`</></right>`)
			screen.Render(1, d.renderer.Render())

			screen.RenderLineWithBackGround(0, screen.Height-1, keyMappings, menuBarBackgroundColor)
			d.State.changed = false
		}
	}

	screen.Flush()
}

//Write sends dry output to the given writer
func Write(d *Dry, w io.Writer) {
	switch d.State.viewMode {
	case StatsMode:
		{
			if d.stats != nil {
				fmt.Fprintf(w, appui.NewDockerStatsRenderer(d.stats).Render())
			} else {
				fmt.Fprintf(w, "Could not read stats")
			}
		}
	case InspectMode:
		{
			fmt.Fprintf(w, appui.NewDockerInspectRenderer(d.containerToInspect).Render())
		}
	case HelpMode:
		{
			fmt.Fprintf(w, help)
		}

	case InfoMode:
		fmt.Fprintf(w, appui.NewDockerInfoRenderer(d.info).Render())
	default:
		{
			fmt.Fprintf(w, "Dry is not ready yet for rendering, be patient...")
		}
	}
}
