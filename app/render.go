package app

import (
	"context"
	"time"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

var cancelMonitorWidget context.CancelFunc

//Render renders dry on the given screen
func Render(d *Dry, screen *ui.Screen, statusBar *ui.StatusBar) {
	var bufferers []gizaktermui.Bufferer

	var count int
	var keymap string
	var viewRenderer ui.Renderer
	di := d.widgetRegistry.DockerInfo
	bufferers = append(bufferers, di)

	//if the monitor widget is active it is now cancelled since (most likely) the view is going to change now
	if cancelMonitorWidget != nil {
		cancelMonitorWidget()
	}

	switch d.viewMode() {
	case Main:
		{
			containersWidget := d.widgetRegistry.ContainerList
			if err := containersWidget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			count = containersWidget.RowCount()
			bufferers = append(bufferers, containersWidget)
			keymap = keyMappings

		}
	case Images:
		{
			//after a refresh, sorting is needed
			sortMode := d.state.sortImagesMode
			widget := d.widgetRegistry.ImageList

			images, err := d.dockerDaemon.Images()
			if err == nil {
				count = len(images)
				data := appui.NewDockerImageRenderData(
					images,
					sortMode)

				widget.PrepareToRender(data)
				bufferers = append(bufferers, widget)

			} else {
				screen.Render(1, err.Error())
			}

			keymap = imagesKeyMappings

		}
	case Networks:
		{
			//after a refresh, sorting is needed
			sortMode := d.state.sortNetworksMode
			widget := d.widgetRegistry.Networks

			networks, err := d.dockerDaemon.Networks()
			if err == nil {
				count = len(networks)
				data := appui.NewDockerNetworkRenderData(
					networks,
					sortMode)

				widget.PrepareToRender(data)
				bufferers = append(bufferers, widget)

			} else {
				screen.Render(1, err.Error())
			}

			keymap = networkKeyMappings

		}
	case Nodes:
		{
			nodes := d.widgetRegistry.Nodes
			nodes.Mount()
			bufferers = append(bufferers, nodes)
			count = nodes.RowCount()
			keymap = swarmMapping
		}
	case Services:
		{
			servicesWidget := d.widgetRegistry.ServiceList
			servicesWidget.Mount()
			bufferers = append(bufferers, servicesWidget)
			count = servicesWidget.RowCount()
			keymap = serviceKeyMappings
		}
	case Tasks:
		{
			tasks := d.widgetRegistry.NodeTasks
			bufferers = append(bufferers, tasks)
			count = tasks.RowCount()
			keymap = swarmMapping
		}
	case ServiceTasks:
		{
			tasks := d.widgetRegistry.ServiceTasks
			count = tasks.RowCount()
			bufferers = append(bufferers, tasks)
			keymap = swarmMapping
		}
	case DiskUsage:
		{
			if du, err := d.dockerDaemon.DiskUsage(); err == nil {
				d.widgetRegistry.DiskUsage.PrepareToRender(&du, d.PruneReport())
				viewRenderer = d.widgetRegistry.DiskUsage

			} else {
				screen.Render(1,
					"There was an error retrieving disk usage information.")
			}
			keymap = diskUsageKeyMappings
		}
	case Monitor:
		{
			monitor := d.widgetRegistry.Monitor
			monitor.Mount()
			ctx, cancel := context.WithCancel(context.Background())
			monitor.RenderLoop(ctx)
			keymap = monitorMapping
			cancelMonitorWidget = cancel
			count = monitor.RowCount()
		}
	}

	updateCursorPosition(screen.Cursor, count)
	bufferers = append(bufferers, footer(keymap))

	statusBar.Render()
	screen.RenderLine(0, 0, `<right><white>`+time.Now().Format(`15:04:05`)+`</></right>`)
	screen.RenderBufferer(bufferers...)
	if viewRenderer != nil {
		screen.RenderRenderer(appui.MainScreenHeaderSize, viewRenderer)
	}

	for _, widget := range d.widgetRegistry.activeWidgets {
		screen.RenderBufferer(widget)
	}

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
	case InspectImageMode:
		output = appui.NewJSONRenderer(d.inspectedImage)
	case InspectNetworkMode:
		output = appui.NewJSONRenderer(d.inspectedNetwork)
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

func footer(mapping string) *termui.MarkupPar {

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
