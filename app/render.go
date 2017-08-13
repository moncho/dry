package app

import (
	"context"
	"fmt"
	"time"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

const (
	viewStartingLine = appui.MainScreenHeaderSize + 2
)

var cancelMonitorWidget context.CancelFunc

//Render renders dry on the given screen
func Render(d *Dry, screen *ui.Screen, statusBar *ui.StatusBar) {
	var bufferers []gizaktermui.Bufferer

	var what string
	var count int
	var titleInfo string
	var keymap string
	var viewRenderer ui.Renderer
	di := d.widgetRegistry.DockerInfo
	bufferers = append(bufferers, di)

	//if the monitor widget is active it is now cancelled since (most likely) the view is going to change now
	if cancelMonitorWidget != nil {
		cancelMonitorWidget()
	}
	d.state.activeWidget = nil

	switch d.viewMode() {
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
			containersWidget := d.widgetRegistry.ContainerList
			containersWidget.PrepareToRender(data)

			d.state.activeWidget = containersWidget
			bufferers = append(bufferers, containersWidget)

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
			renderer := d.widgetRegistry.ImageList

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
			nodes := swarm.NewNodesWidget(d.dockerDaemon, viewStartingLine)
			d.state.activeWidget = nodes
			bufferers = append(bufferers, nodes)
			what = "Nodes"
			count = nodes.RowCount()
			updateCursorPosition(screen.Cursor, count)
			keymap = swarmMapping

		}
	case Services:
		{
			services := swarm.NewServicesWidget(d.dockerDaemon, viewStartingLine)
			d.state.activeWidget = services
			bufferers = append(bufferers, services)
			what = "Services"
			count = services.RowCount()
			updateCursorPosition(screen.Cursor, count)
			keymap = swarmMapping
		}
	case Tasks:
		{
			nodeID := d.state.node
			tasks := swarm.NewNodeTasksWidget(d.dockerDaemon, nodeID, viewStartingLine)
			bufferers = append(bufferers, tasks)
			whatNode := nodeID
			if node, err := d.dockerDaemon.ResolveNode(nodeID); err == nil {
				whatNode = node
			}
			what = fmt.Sprintf("Node %s tasks", whatNode)
			count = tasks.RowCount()
			updateCursorPosition(screen.Cursor, count)
			keymap = swarmMapping
		}
	case ServiceTasks:
		{
			serviceID := d.state.service
			tasks := swarm.NewServiceTasksWidget(d.dockerDaemon, serviceID, appui.MainScreenHeaderSize)
			bufferers = append(bufferers, tasks)
			updateCursorPosition(screen.Cursor, tasks.RowCount())
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

	for _, widget := range d.widgetRegistry.activeWidgets {
		screen.RenderBufferer(widget)
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
