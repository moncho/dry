package appui

import "github.com/moncho/dry/ui/termui"

// EventCommand type alias for a func to be run by an EventableWidget
type EventCommand func(string) error

// EventableWidget interface defines how widgets receive events
type EventableWidget interface {
	OnEvent(EventCommand) error
}

// FilterableWidget interface defines how widgets filter
type FilterableWidget interface {
	Filter(filter string)
}

// SortableWidget interface defines how widgets sort
type SortableWidget interface {
	Sort()
}

// AppWidget groups common behaviour for appui widgets
type AppWidget interface {
	termui.Widget
	EventableWidget
	FilterableWidget
	SortableWidget
}
