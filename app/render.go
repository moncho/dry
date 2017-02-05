package app

import (
	"fmt"
	"time"

	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
)

//ViewMode represents dry possible views
type viewMode uint16

//known view modes
const (
	Main viewMode = iota //This is the container list view
	DiskUsage
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
			containers := d.containerList()

			count = len(containers)
			updateCursorPosition(screen.Cursor, count)
			data := appui.NewDockerPsRenderData(
				containers,
				screen.Cursor.Position(),
				sortMode)
			d.ui.ContainerComponent.PrepareToRender(data)
			screen.Render(1, d.ui.ContainerComponent.Render())

			keymap = keyMappings
			title := fmt.Sprintf(
				"<b><blue>Containers: </><yellow>%d</></>", count)
			if d.state.filterPattern != "" {
				title = title + fmt.Sprintf(
					"<b><blue> | Showing containers named: </><yellow>%s</></> ", d.state.filterPattern)
			}

			screen.RenderLine(0, appui.MainScreenHeaderSize, title)

		}
	case Images:
		{
			//after a refresh, sorting is needed
			sortMode := d.state.SortImagesMode
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
			renderViewTitle(screen, what, count)

		}
	case Networks:
		{
			screen.Render(1,
				appui.NewDockerNetworksRenderer(d.dockerDaemon, screen.Height, screen.Cursor, d.state.SortNetworksMode).Render())
			what = "Networks"
			count = d.dockerDaemon.NetworksCount()
			updateCursorPosition(screen.Cursor, count)
			keymap = networkKeyMappings
			renderViewTitle(screen, what, count)

		}

	}
	screen.RenderLineWithBackGround(0, screen.Height-1, keymap, appui.DryTheme.Footer)
	d.setChanged(false)

	screen.Flush()
}

//renderDry returns a Renderer with dry's current content
func renderDry(d *Dry) ui.Renderer {
	var output ui.Renderer
	switch d.viewMode() {
	case EventsMode:
		output = appui.NewDockerEventsRenderer(d.dockerDaemon.EventLog().Events())
	case ImageHistoryMode:
		output = appui.NewDockerImageHistoryRenderer(d.imageHistory)
	case InspectMode:
		output = appui.NewDockerInspectRenderer(d.inspectedContainer)
	case InspectImageMode:
		output = appui.NewDockerInspectImageRenderer(d.inspectedImage)
	case InspectNetworkMode:
		output = appui.NewDockerInspectNetworkRenderer(d.inspectedNetwork)
	case HelpMode:
		output = ui.StringRenderer(help)
	case InfoMode:
		output = appui.NewDockerInfoRenderer(d.info)
	default:
		{
			output = ui.StringRenderer("Dry is not ready yet for rendering, be patient...")
		}
	}
	return output
}

func renderViewTitle(screen *ui.Screen, what string, howMany int) {
	screen.RenderLine(0, appui.MainScreenHeaderSize,
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
