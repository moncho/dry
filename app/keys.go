package app

import "charm.land/bubbles/v2/key"

type globalKeyMap struct {
	Quit         key.Binding
	Help         key.Binding
	Containers   key.Binding
	Images       key.Binding
	Networks     key.Binding
	Volumes      key.Binding
	Nodes        key.Binding
	Services     key.Binding
	Stacks       key.Binding
	Monitor      key.Binding
	ToggleHeader key.Binding
	DiskUsage    key.Binding
	Events       key.Binding
	DockerInfo   key.Binding
}

var globalKeys = globalKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("Q", "ctrl+c"),
		key.WithHelp("Q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?", "h", "H"),
		key.WithHelp("h", "help"),
	),
	Containers: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "containers"),
	),
	Images: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "images"),
	),
	Networks: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "networks"),
	),
	Volumes: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "volumes"),
	),
	Nodes: key.NewBinding(
		key.WithKeys("5"),
		key.WithHelp("5", "nodes"),
	),
	Services: key.NewBinding(
		key.WithKeys("6"),
		key.WithHelp("6", "services"),
	),
	Stacks: key.NewBinding(
		key.WithKeys("7"),
		key.WithHelp("7", "stacks"),
	),
	Monitor: key.NewBinding(
		key.WithKeys("m", "M"),
		key.WithHelp("m", "monitor"),
	),
	ToggleHeader: key.NewBinding(
		key.WithKeys("f7"),
		key.WithHelp("F7", "toggle header"),
	),
	DiskUsage: key.NewBinding(
		key.WithKeys("f8"),
		key.WithHelp("F8", "disk usage"),
	),
	Events: key.NewBinding(
		key.WithKeys("f9"),
		key.WithHelp("F9", "events"),
	),
	DockerInfo: key.NewBinding(
		key.WithKeys("f10"),
		key.WithHelp("F10", "docker info"),
	),
}
