package app

import (
	"fmt"

	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/ui/termui"
)

// render renders dry on the given screen
func render(d *Dry, screen *ui.Screen) {

	var bufferers []gizaktermui.Bufferer

	if d.showingHeader() {
		bufferers = append(bufferers, widgets.DockerInfo)
	}

	var keymap string
	var viewRenderer fmt.Stringer

	switch d.viewMode() {
	case ContainerMenu:
		{
			cMenu := widgets.ContainerMenu
			if err := cMenu.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, cMenu)

			keymap = commandsMenuBar

		}
	case Main:
		{
			containersWidget := widgets.ContainerList
			if err := containersWidget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, containersWidget)
			keymap = keyMappings

		}
	case Images:
		{

			widget := widgets.ImageList
			if err := widget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, widget)

			keymap = imagesKeyMappings

		}
	case Networks:
		{
			widget := widgets.Networks
			if err := widget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
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
			keymap = nodeKeyMappings
		}
	case Services:
		{
			servicesWidget := widgets.ServiceList
			if err := servicesWidget.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, servicesWidget)
			keymap = serviceKeyMappings
		}
	case Tasks:
		{
			tasks := widgets.NodeTasks
			if err := tasks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, tasks)
			keymap = swarmMapping
		}
	case ServiceTasks:
		{
			tasks := widgets.ServiceTasks
			if err := tasks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, tasks)
			keymap = swarmMapping
		}
	case Stacks:
		{
			stacks := widgets.Stacks
			if err := stacks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, stacks)
			keymap = stackKeyMappings
		}
	case StackTasks:
		{
			tasks := widgets.StackTasks
			if err := tasks.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, tasks)
			keymap = swarmMapping
		}
	case DiskUsage:
		{
			viewRenderer = widgets.DiskUsage
			keymap = diskUsageKeyMappings
		}
	case Monitor:
		{
			monitor := widgets.Monitor
			monitor.Mount()
			keymap = monitorMapping
		}
	case Volumes:
		{
			volumes := widgets.Volumes
			if err := volumes.Mount(); err != nil {
				screen.Render(1, err.Error())
			}
			bufferers = append(bufferers, volumes)
			keymap = volumesKeyMappings
		}

	}
	bufferers = append(bufferers, footer(keymap))

	widgets.MessageBar.Render()
	screen.RenderBufferer(bufferers...)
	if viewRenderer != nil {
		screen.Render(appui.MainScreenHeaderSize, viewRenderer.String())
	}

	for _, widget := range widgets.activeWidgets() {
		screen.RenderBufferer(widget)
	}

	screen.Flush()
}

func footer(mapping string) *termui.MarkupPar {

	d := ui.ActiveScreen.Dimensions()
	par := termui.NewParFromMarkupText(appui.DryTheme, mapping)
	par.SetX(0)
	par.SetY(d.Height - 1)
	par.Border = false
	par.Width = d.Width
	par.TextBgColor = gizaktermui.Attribute(appui.DryTheme.Footer)
	par.Bg = gizaktermui.Attribute(appui.DryTheme.Footer)

	return par
}

// Updates the cursor position in case it is out of bounds
func updateCursorPosition(cursor *ui.Cursor, noOfElements int) {
	cursor.Max(noOfElements - 1)
}
