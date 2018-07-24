package app

import (
	"context"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

var cancelMonitorWidget context.CancelFunc

//render renders dry on the given screen
func render(d *Dry, screen *ui.Screen) {
	var bufferers []gizaktermui.Bufferer

	var count int
	var keymap string
	var viewRenderer ui.Renderer
	di := widgets.DockerInfo
	bufferers = append(bufferers, di)

	switch d.viewMode() {
	case ContainerMenu:
		{
			cMenu := widgets.ContainerMenu
			if err := cMenu.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, cMenu)
			count = cMenu.RowCount()

			keymap = commandsMenuBar

		}
	case Main:
		{
			containersWidget := widgets.ContainerList
			if err := containersWidget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			count = containersWidget.RowCount()
			bufferers = append(bufferers, containersWidget)
			keymap = keyMappings

		}
	case Images:
		{

			widget := widgets.ImageList
			if err := widget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			count = widget.RowCount()
			bufferers = append(bufferers, widget)

			keymap = imagesKeyMappings

		}
	case Networks:
		{
			widget := widgets.Networks
			if err := widget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			count = widget.RowCount()
			bufferers = append(bufferers, widget)
			keymap = networkKeyMappings
		}
	case Nodes:
		{
			nodes := widgets.Nodes
			if err := nodes.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, nodes)
			count = nodes.RowCount()
			keymap = nodeKeyMappings
		}
	case Services:
		{
			servicesWidget := widgets.ServiceList
			if err := servicesWidget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, servicesWidget)
			count = servicesWidget.RowCount()
			keymap = serviceKeyMappings
		}
	case Tasks:
		{
			tasks := widgets.NodeTasks
			if err := tasks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, tasks)
			count = tasks.RowCount()
			keymap = swarmMapping
		}
	case ServiceTasks:
		{
			tasks := widgets.ServiceTasks
			if err := tasks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, tasks)
			count = tasks.RowCount()
			keymap = swarmMapping
		}
	case Stacks:
		{
			stacks := widgets.Stacks
			if err := stacks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, stacks)
			count = stacks.RowCount()
			keymap = stackKeyMappings
		}
	case StackTasks:
		{
			tasks := widgets.StackTasks
			if err := tasks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, tasks)
			count = tasks.RowCount()
			keymap = swarmMapping
		}
	case DiskUsage:
		{
			viewRenderer = widgets.DiskUsage
			keymap = diskUsageKeyMappings
		}
	case Monitor:
		{
			if cancelMonitorWidget != nil {
				cancelMonitorWidget()
			}
			monitor := widgets.Monitor
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

	widgets.MessageBar.Render()
	screen.RenderBufferer(bufferers...)
	if viewRenderer != nil {
		screen.RenderRenderer(appui.MainScreenHeaderSize, viewRenderer)
	}

	for _, widget := range widgets.activeWidgets {
		screen.RenderBufferer(widget)
	}

	screen.Flush()
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
