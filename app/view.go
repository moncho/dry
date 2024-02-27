package app

// viewMode represents dry possible views
type viewMode uint16

// existing view modes
const (
	Main viewMode = iota //This is the container list view
	DiskUsage
	Images
	Monitor
	Networks
	EventsMode
	HelpMode
	InfoMode
	Nodes
	Services
	ServiceTasks
	Stacks
	StackTasks
	Tasks
	ContainerMenu
	Volumes
	NoView
)
