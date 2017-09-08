package appui

import "github.com/moncho/dry/ui/termui"

//EventCommand type alias for a func to be run by an Actionable
type EventCommand func(string) error

//Actionable interface describes how widgets run commands as a result of an event
type Actionable interface {
	termui.Widget
	OnEvent(EventCommand) error
}
