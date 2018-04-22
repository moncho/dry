package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/swarm"
)

//Container holds a detailed view of a container
type Container struct {
	types.Container
	types.ContainerJSON
}

//ContainerDaemon describes what is expected from the container daemon
type ContainerDaemon interface {
	ContainerAPI
	ImageAPI
	NetworkAPI
	SwarmAPI
	DiskUsage() (types.DiskUsage, error)
	DockerEnv() *Env
	Events() (<-chan events.Message, chan<- struct{}, error)
	EventLog() *EventLog
	Info() (types.Info, error)
	InspectImage(id string) (types.ImageInspect, error)
	Ok() (bool, error)
	Prune() (*PruneReport, error)
	Rm(id string) error
	Rmi(id string, force bool) ([]types.ImageDeleteResponseItem, error)
	Refresh(notify func(error))
	RemoveDanglingImages() (int, error)
	RemoveNetwork(id string) error
	Version() (*types.Version, error)
}

//ContainerAPI defines the API for containers
type ContainerAPI interface {
	ContainerByID(id string) *Container
	Containers(filter []ContainerFilter, mode SortMode) []*Container
	Inspect(id string) (types.ContainerJSON, error)
	IsContainerRunning(id string) bool
	Kill(id string) error
	Logs(id string, since string) (io.ReadCloser, error)
	OpenChannel(container *Container) *StatsChannel
	RemoveAllStoppedContainers() (int, error)
	RestartContainer(id string) error
	StopContainer(id string) error
	Top(id string) (container.ContainerTopOKBody, error)
}

//ImageAPI defines the API for Docker images
type ImageAPI interface {
	History(id string) ([]image.HistoryResponseItem, error)
	ImageByID(id string) (types.ImageSummary, error)
	Images() ([]types.ImageSummary, error)
	RunImage(image types.ImageSummary, command string) error
}

//NetworkAPI defines the API for Docker networks
type NetworkAPI interface {
	Networks() ([]types.NetworkResource, error)
	NetworkInspect(id string) (types.NetworkResource, error)
}

//SwarmAPI defines the API for Docker Swarm
type SwarmAPI interface {
	Node(id string) (*swarm.Node, error)
	NodeChangeAvailability(nodeID string, availability swarm.NodeAvailability) error
	Nodes() ([]swarm.Node, error)
	NodeTasks(nodeID string) ([]swarm.Task, error)
	ResolveNode(id string) (string, error)
	ResolveService(id string) (string, error)
	Service(id string) (*swarm.Service, error)
	ServiceLogs(id string, since string) (io.ReadCloser, error)
	Services() ([]swarm.Service, error)
	ServiceRemove(id string) error
	ServiceScale(id string, replicas uint64) error
	ServiceTasks(services ...string) ([]swarm.Task, error)
	Stacks() ([]Stack, error)
	StackConfigs(stack string) ([]swarm.Config, error)
	StackNetworks(stack string) ([]types.NetworkResource, error)
	StackRemove(id string) error
	StackSecrets(stack string) ([]swarm.Secret, error)
	StackTasks(stack string) ([]swarm.Task, error)
	Task(id string) (swarm.Task, error)
}

//Stats holds runtime stats for a container
type Stats struct {
	CID              string
	Command          string
	CPUPercentage    float64
	Memory           float64
	MemoryLimit      float64
	MemoryPercentage float64
	NetworkRx        float64
	NetworkTx        float64
	BlockRead        float64
	BlockWrite       float64
	PidsCurrent      uint64
	Stats            *types.StatsJSON
	ProcessList      *container.ContainerTopOKBody
}

//Resolver defines the interface for ID to name resolution
type Resolver interface {
	Resolve(ctx context.Context, t interface{}, id string) (string, error)
}
