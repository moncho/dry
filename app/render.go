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
func Render(d *Dry, screen *ui.Screen, statusBar *ui.ExpiringMessageWidget) {
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
	case ContainerMenu:
		{
			cMenu := d.widgetRegistry.ContainerMenu
			if err := cMenu.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, cMenu)
			count = cMenu.RowCount()

			keymap = commandsMenuBar

		}
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

			widget := d.widgetRegistry.ImageList
			if err := widget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			count = widget.RowCount()
			bufferers = append(bufferers, widget)

			keymap = imagesKeyMappings

		}
	case Networks:
		{
			widget := d.widgetRegistry.Networks
			if err := widget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			count = widget.RowCount()
			bufferers = append(bufferers, widget)
			keymap = networkKeyMappings
		}
	case Nodes:
		{
			nodes := d.widgetRegistry.Nodes
			if err := nodes.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, nodes)
			count = nodes.RowCount()
			keymap = nodeKeyMappings
		}
	case Services:
		{
			servicesWidget := d.widgetRegistry.ServiceList
			if err := servicesWidget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, servicesWidget)
			count = servicesWidget.RowCount()
			keymap = serviceKeyMappings
		}
	case Tasks:
		{
			tasks := d.widgetRegistry.NodeTasks
			if err := tasks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, tasks)
			count = tasks.RowCount()
			keymap = swarmMapping
		}
	case ServiceTasks:
		{
			tasks := d.widgetRegistry.ServiceTasks
			if err := tasks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, tasks)
			count = tasks.RowCount()
			keymap = swarmMapping
		}
	case Stacks:
		{
			stacks := d.widgetRegistry.Stacks
			if err := stacks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, stacks)
			count = stacks.RowCount()
			keymap = stackKeyMappings
		}
	case StackTasks:
		{
			tasks := d.widgetRegistry.StackTasks
			if err := tasks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, tasks)
			count = tasks.RowCount()
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
}
