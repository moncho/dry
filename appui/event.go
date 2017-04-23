package appui

type EventHandler func(string) error

type Actionable interface {
	OnEvent(EventHandler) error
}
