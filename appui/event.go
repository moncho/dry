package appui

//EventCommand type alias for a func to be run by an Actionable
type EventCommand func(string) error

//Actionable interface describes how widgets run commands as a result of an event
type Actionable interface {
	OnEvent(EventCommand) error
}
