package app

//ViewMode represents dry possible views
type viewMode uint16

//known view modes
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
)

//isMainScreen returns true if this viewMode is one of the main screens of dry
func (v viewMode) isMainScreen() bool {
	switch v {
	case Main, Networks, Images, Monitor, Nodes, Services, Stacks:
		return true
	default:
		return false
	}
}
