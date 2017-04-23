package app

import (
	"context"
	"fmt"
	"time"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

//ViewMode represents dry possible views
type viewMode uint16

//known view modes
const (
	Main viewMode = iota //This is the container list view
	DiskUsage
	Images
	Monitor
	Networks
	EventsMode
	HelpMode
	ImageHistoryMode
	InfoMode
	InspectImageMode
	InspectNetworkMode
	InspectMode
	Nodes
	Services
	Tasks
)

const (
	viewStartingLine = appui.MainScreenHeaderSize + 2
)

var cancelMonitorWidget context.CancelFunc

//Render renders dry in the given screen
func Render(d *Dry, screen *ui.Screen, statusBar *ui.StatusBar) {
	var bufferers []gizaktermui.Bufferer

	var what string
	var count int
	var titleInfo string
	var keymap string
	var viewRenderer ui.Renderer
	di := d.ui.DockerInfoWidget
	bufferers = append(bufferers, di)

	viewMode := d.viewMode()
	//if the monitor widget is active it is now cancelled since (most likely) the view is going to change now
	if cancelMonitorWidget != nil {
		cancelMonitorWidget()
	}
	d.state.activeWidget = nil

	switch viewMode {
	case Main:
		{
			//after a refresh, sorting is needed
			sortMode := d.state.sortMode
			containers := d.containerList()

			count = len(containers)
			updateCursorPosition(screen.Cursor, count)
			data := appui.NewDockerPsRenderData(
				containers,
				sortMode)
			d.ui.ContainerListWidget.PrepareToRender(data)
			viewRenderer = d.ui.ContainerListWidget

			keymap = keyMappings

			if d.state.filterPattern != "" {
				titleInfo = titleInfo + fmt.Sprintf(
					"<b><blue> | Container name filter: </><yellow>%s</></> ", d.state.filterPattern)
			}
			what = "Containers"

		}
	case Images:
		{
			//after a refresh, sorting is needed
			sortMode := d.state.sortImagesMode
			renderer := d.ui.ImageListWidget

			images, err := d.dockerDaemon.Images()
			if err == nil {
				count = len(images)
				updateCursorPosition(screen.Cursor, count)
				data := appui.NewDockerImageRenderData(
					images,
					sortMode)

				renderer.PrepareForRender(data)
				viewRenderer = renderer
			} else {
				screen.Render(1, err.Error())
			}

			what = "Images"
			keymap = imagesKeyMappings

		}
	case Networks:
		{
			viewRenderer = appui.NewDockerNetworksRenderer(d.dockerDaemon, screen.Cursor, d.state.sortNetworksMode)
			what = "Networks"
			count = d.dockerDaemon.NetworksCount()
			updateCursorPosition(screen.Cursor, count)
			keymap = networkKeyMappings

		}
	case Nodes:
		{
			nodes := appui.NewSwarmNodesWidget(d.dockerDaemon, viewStartingLine)
			d.state.activeWidget = nodes
			bufferers = append(bufferers, nodes)
			what = "Nodes"
			count = nodes.RowCount()
			updateCursorPosition(screen.Cursor, count)
			keymap = swarmMapping

		}
	case Services:
		{
			nodes := appui.NewSwarmNodesWidget(d.dockerDaemon, viewStartingLine)
			bufferers = append(bufferers, nodes)
			what = "Services"
			count = nodes.RowCount()
			updateCursorPosition(screen.Cursor, count)
			keymap = swarmMapping
		}
	case Tasks:
		{
			tasks := appui.NewTasksWidget(d.dockerDaemon, d.state.node, viewStartingLine)
			bufferers = append(bufferers, tasks)
			what = "Tasks"
			count = tasks.RowCount()
			updateCursorPosition(screen.Cursor, count)
			keymap = swarmMapping
		}
	case DiskUsage:
		{
			if du, err := d.dockerDaemon.DiskUsage(); err == nil {
				d.ui.DiskUsageWidget.PrepareToRender(&du, d.PruneReport())
				viewRenderer = d.ui.DiskUsageWidget

			} else {
				screen.Render(1,
					"There was an error retrieving disk usage information.")
			}
			keymap = diskUsageKeyMappings
		}
	case Monitor:
		{
			monitor := appui.NewMonitor(d.dockerDaemon, viewStartingLine)
			ctx, cancel := context.WithCancel(context.Background())
			monitor.RenderLoop(ctx)
			keymap = monitorMapping
			what = "Containers"
			count = monitor.RowCount()
			cancelMonitorWidget = cancel
			updateCursorPosition(screen.Cursor, count)
		}
	}

	if what != "" {
		bufferers = append(bufferers, tableHeader(screen, what, count, titleInfo))
	}

	bufferers = append(bufferers, footer(screen, keymap))

	statusBar.Render()
	screen.RenderLine(0, 0, `<right><white>`+time.Now().Format(`15:04:05`)+`</></right>`)
	screen.RenderBufferer(bufferers...)
	if viewRenderer != nil {
		screen.RenderRenderer(viewStartingLine, viewRenderer)
	}

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

func tableHeader(screen *ui.Screen, what string, howMany int, info string) *termui.MarkupPar {
	par := termui.NewParFromMarkupText(appui.DryTheme,
		fmt.Sprintf(
			"<b><blue>%s: </><yellow>%d</></>", what, howMany)+" "+info)

	par.SetX(0)
	par.SetY(appui.MainScreenHeaderSize)
	par.Border = false
	par.Width = ui.ActiveScreen.Dimensions.Width
	par.TextBgColor = gizaktermui.Attribute(appui.DryTheme.Bg)
	par.Bg = gizaktermui.Attribute(appui.DryTheme.Bg)

	return par
}

func footer(screen *ui.Screen, mapping string) *termui.MarkupPar {

	par := termui.NewParFromMarkupText(appui.DryTheme, mapping)
	par.SetX(0)
	par.SetY(ui.ActiveScreen.Dimensions.Height - 1)
	par.Border = false
	par.Width = ui.ActiveScreen.Dimensions.Width
	par.TextBgColor = gizaktermui.Attribute(appui.DryTheme.Footer)
	par.Bg = gizaktermui.Attribute(appui.DryTheme.Footer)

	return par
}

//Updates the cursor position in case it is out of bounds
func updateCursorPosition(cursor *ui.Cursor, noOfElements int) {
	cursor.Max(noOfElements - 1)
	if cursor.Position() >= noOfElements {
		cursor.Bottom()
	} else if cursor.Position() < 0 {
		cursor.Reset()
	}
}
