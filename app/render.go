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
)

//Render knows how to render dry app in the given screen
func Render(d *Dry, screen *ui.Screen) {
	switch d.State.viewMode {
	case Main:
		{
			//after a refresh, sorting is needed
			d.dockerDaemon.Refresh(d.State.showingAllContainers)
			d.dockerDaemon.Sort(d.State.SortMode)
			d.renderer.SortMode(d.State.SortMode)
			screen.Render(0, d.renderer.Render())
			screen.RenderLine(0, 0, `<right><white>`+time.Now().Format(`3:04:05pm PST`)+`</></right>`)
			screen.RenderLine(0, screen.Height-1, keyMappings)
			d.State.changed = false
		}
	case HelpMode:
		{
			screen.Clear()
			screen.Render(0, help)
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
			}
		}
	case InspectMode:
		{
			fmt.Fprintf(w, appui.NewDockerInspectRenderer(d.containerToInspect).Render())
		}
	default:
		{
			fmt.Fprintf(w, "Dry is not ready yet for rendering, be patient...")
		}
	}
}
