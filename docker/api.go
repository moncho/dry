package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/volume"
)

// Container holds a detailed view of a container
type Container struct {
	types.Container
	types.ContainerJSON
}

// ContainerDaemon describes what is expected from the container daemon
type ContainerDaemon interface {
	ContainerAPI
	ImageAPI
	NetworkAPI
	VolumesAPI
	SwarmAPI
	ContainerRuntime
	DiskUsage() (types.DiskUsage, error)
	DockerEnv() Env
	Events() (<-chan events.Message, chan<- struct{}, error)
	EventLog() *EventLog
	Info() (types.Info, error)
	InspectImage(id string) (types.ImageInspect, error)
	Ok() (bool, error)
	Prune() (*PruneReport, error)
	Rm(id string) error
	Refresh(notify func(error))
	RemoveNetwork(id string) error
	Version() (*types.Version, error)
}

// ContainerAPI is a subset of the Docker API to manage containers
type ContainerAPI interface {
	ContainerByID(id string) *Container
	Containers(filter []ContainerFilter, mode SortMode) []*Container
	Inspect(id string) (types.ContainerJSON, error)
	IsContainerRunning(id string) bool
	Kill(id string) error
	Logs(id string, since string, withTimeStamp bool) (io.ReadCloser, error)
	RemoveAllStoppedContainers() (int, error)
	RestartContainer(id string) error
	StopContainer(id string) error
}

// ContainerRuntime is the subset of the Docker API to query container runtime information
type ContainerRuntime interface {
	StatsChannel(container *Container) (*StatsChannel, error)
	Top(ctx context.Context, id string) (container.ContainerTopOKBody, error)
}

// ImageAPI is a subset of the Docker API to manage images
type ImageAPI interface {
	History(id string) ([]image.HistoryResponseItem, error)
	ImageByID(id string) (types.ImageSummary, error)
	Images() ([]types.ImageSummary, error)
	RemoveDanglingImages() (int, error)
	RemoveUnusedImages() (int, error)
	Rmi(id string, force bool) ([]types.ImageDeleteResponseItem, error)
	RunImage(image types.ImageSummary, command string) error
}

// NetworkAPI is a subset of the Docker API to manage networks
type NetworkAPI interface {
	Networks() ([]types.NetworkResource, error)
	NetworkInspect(id string) (types.NetworkResource, error)
}

// SwarmAPI defines the API for Docker Swarm
type SwarmAPI interface {
	Node(id string) (*swarm.Node, error)
	NodeChangeAvailability(nodeID string, availability swarm.NodeAvailability) error
	Nodes() ([]swarm.Node, error)
	NodeTasks(nodeID string) ([]swarm.Task, error)
	ResolveNode(id string) (string, error)
	ResolveService(id string) (string, error)
	Service(id string) (*swarm.Service, error)
	ServiceLogs(id string, since string, withTimeStamps bool) (io.ReadCloser, error)
	Services() ([]swarm.Service, error)
	ServiceRemove(id string) error
	ServiceScale(id string, replicas uint64) error
	ServiceTasks(services ...string) ([]swarm.Task, error)
	ServiceUpdate(id string) error
	Stacks() ([]Stack, error)
	StackConfigs(stack string) ([]swarm.Config, error)
	StackNetworks(stack string) ([]types.NetworkResource, error)
	StackRemove(id string) error
	StackSecrets(stack string) ([]swarm.Secret, error)
	StackTasks(stack string) ([]swarm.Task, error)
	Task(id string) (swarm.Task, error)
}

// Stats holds runtime stats for a container
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
	Error            error
}

// Resolver defines the interface for ID to name resolution
type Resolver interface {
	Resolve(ctx context.Context, t interface{}, id string) (string, error)
}

// VolumesAPI defines the API for Docker volumes.
type VolumesAPI interface {
	VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error)
	VolumeList(ctx context.Context) ([]*volume.Volume, error)
	VolumePrune(ctx context.Context) (int, error)
	VolumeRemove(ctx context.Context, volumeID string, force bool) error
	VolumeRemoveAll(ctx context.Context) (int, error)
}
