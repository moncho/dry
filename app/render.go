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
	Images
	Networks
	EventsMode
	HelpMode
	ImageHistoryMode
	InfoMode
	InspectImageMode
	InspectNetworkMode
	InspectMode
)

const (
	//The position from the top (0) where a line describing what is
	//being shown is placed. Kind of a magic number.
	screenDescriptionIndex = 5
)

//Render renders dry in the given screen
func Render(d *Dry, screen *ui.Screen, statusBar *ui.StatusBar) {
	var what string
	var count int
	var keymap string
	statusBar.Render()
	screen.RenderLine(0, 0, `<right><white>`+time.Now().Format(`15:04:05`)+`</></right>`)
	switch d.viewMode() {
	case Main:
		{
			//after a refresh, sorting is needed
			sortMode := d.state.SortMode
			//d.dockerDaemon.Sort(sortMode)
			containers := d.dockerDaemon.Containers()
			count = len(containers)
			updateCursorPosition(screen.Cursor, count)
			data := appui.NewDockerPsRenderData(
				containers,
				screen.Cursor.Position(),
				sortMode)
			d.renderer.PrepareToRender(data)
			screen.Render(1, d.renderer.Render())

			what = "Containers"
			keymap = keyMappings

		}
	case Images:
		{
			//after a refresh, sorting is needed
			sortMode := d.state.SortImagesMode
			//d.dockerDaemon.SortImages(sortMode)
			renderer := appui.NewDockerImagesRenderer(d.dockerDaemon, screen.Height)

			images, err := d.dockerDaemon.Images()
			if err == nil {
				count = len(images)
				updateCursorPosition(screen.Cursor, count)
				data := appui.NewDockerImageRenderData(
					images,
					screen.Cursor.Position(),
					sortMode)

				renderer.PrepareForRender(data)
				screen.Render(1,
					renderer.Render())
			} else {
				screen.Render(1, err.Error())
			}

			what = "Images"
			keymap = imagesKeyMappings
		}
	case Networks:
		{
			screen.Render(1,
				appui.NewDockerNetworksRenderer(d.dockerDaemon, screen.Height, screen.Cursor, d.state.SortNetworksMode).Render())
			what = "Networks"
			count = d.dockerDaemon.NetworksCount()
			keymap = networkKeyMappings
		}

	}
	renderViewTitle(screen, what, count)
	screen.RenderLineWithBackGround(0, screen.Height-1, keymap, ui.MenuBarBackgroundColor)
	d.setChanged(false)

	screen.Flush()
}

//Write sends dry output to the given writer
func Write(d *Dry, w io.Writer) {
	switch d.viewMode() {
	case EventsMode:
		io.WriteString(w, appui.NewDockerEventsRenderer(d.dockerDaemon.EventLog().Events()).Render())
	case ImageHistoryMode:
		io.WriteString(w, appui.NewDockerImageHistoryRenderer(d.imageHistory).Render())
	case InspectMode:
		io.WriteString(w, appui.NewDockerInspectRenderer(d.inspectedContainer).Render())
	case InspectImageMode:
		io.WriteString(w, appui.NewDockerInspectImageRenderer(d.inspectedImage).Render())
	case InspectNetworkMode:
		io.WriteString(w, appui.NewDockerInspectNetworkRenderer(d.inspectedNetwork).Render())
	case HelpMode:
		io.WriteString(w, help)
	case InfoMode:
		io.WriteString(w, appui.NewDockerInfoRenderer(d.info).Render())
	default:
		{
			io.WriteString(w, "Dry is not ready yet for rendering, be patient...")
		}
	}
}

func renderViewTitle(screen *ui.Screen, what string, howMany int) {
	screen.RenderLine(0, screenDescriptionIndex,
		fmt.Sprintf(
			"<b><blue>%s: </><yellow>%d</></>", what, howMany))
}

//Updates the cursor position in case it is out of bounds
func updateCursorPosition(cursor *ui.Cursor, noOfElements int) {
	if cursor.Position() >= noOfElements {
		cursor.ScrollTo(noOfElements - 1)
	} else if cursor.Position() < 0 {
		cursor.Reset()
	}
}
