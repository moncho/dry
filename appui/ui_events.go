package appui

import "github.com/moncho/dry/ui/termui"

//EventCommand type alias for a func to be run by an Actionable
type EventCommand func(string) error

//EventableWidget interface defines how widgets receive events
type EventableWidget interface {
	termui.Widget
	OnEvent(EventCommand) error
}
