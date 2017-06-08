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
	ImageHistoryMode
	InfoMode
	InspectImageMode
	InspectNetworkMode
	InspectMode
	Nodes
	Services
	ServiceTasks
	Tasks
)

//isMainScreen returns true if this is one of the main screens of dry
func (v viewMode) isMainScreen() bool {
	return v == Main || v == Networks || v == Images || v == Monitor || v == Services
}
