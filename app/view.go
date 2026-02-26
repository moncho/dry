package app

// viewMode represents a view/screen in the application.
type viewMode uint16

const (
	// Main is the container list view
	Main viewMode = iota
	// DiskUsage shows Docker disk usage
	DiskUsage
	// Images is the image list view
	Images
	// Monitor shows real-time container stats
	Monitor
	// Networks is the network list view
	Networks
	// Nodes is the swarm node list view
	Nodes
	// Services is the swarm service list view
	Services
	// ServiceTasks shows tasks for a service
	ServiceTasks
	// Stacks is the swarm stack list view
	Stacks
	// StackTasks shows tasks for a stack
	StackTasks
	// Tasks shows swarm tasks
	Tasks
	// Volumes is the volume list view
	Volumes
)
