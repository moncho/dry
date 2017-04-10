package docker

import (
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
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
	DiskUsage() (types.DiskUsage, error)
	DockerEnv() *Env
	Events() (<-chan events.Message, chan<- struct{}, error)
	EventLog() *EventLog
	History(id string) ([]types.ImageHistory, error)
	ImageAt(pos int) (*types.ImageSummary, error)
	Images() ([]types.ImageSummary, error)
	ImagesCount() int
	Info() (types.Info, error)
	InspectImage(id string) (types.ImageInspect, error)
	Networks() ([]types.NetworkResource, error)
	NetworkAt(pos int) (*types.NetworkResource, error)
	NetworksCount() int
	NetworkInspect(id string) (types.NetworkResource, error)
	Ok() (bool, error)
	Prune() (*PruneReport, error)
	Rm(id string) error
	Rmi(id string, force bool) ([]types.ImageDelete, error)
	Refresh(notify func(error))
	RefreshImages() error
	RefreshNetworks() error
	RemoveDanglingImages() (int, error)
	RemoveNetwork(id string) error
	SortImages(sortMode SortImagesMode)
	SortNetworks(sortMode SortNetworksMode)
	Version() (*types.Version, error)
}

//ContainerAPI defines the API for containers
type ContainerAPI interface {
	ContainerByID(id string) *Container
	Containers(filter ContainerFilter, mode SortMode) []*Container
	Inspect(id string) (types.ContainerJSON, error)
	IsContainerRunning(id string) bool
	Kill(id string) error
	Logs(id string) io.ReadCloser
	OpenChannel(container *Container) *StatsChannel
	RemoveAllStoppedContainers() (int, error)
	RestartContainer(id string) error
	StopContainer(id string) error
	Top(id string) (types.ContainerProcessList, error)
}

//SwarmAPI defines the API for Docker Swarm
type SwarmAPI interface {
	SwarmNodes() ([]swarm.Node, error)
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
	ProcessList      *types.ContainerProcessList
}

//PruneReport represents the result of a prune operation
type PruneReport struct {
	ContainerReport types.ContainersPruneReport
	ImagesReport    types.ImagesPruneReport
	NetworksReport  types.NetworksPruneReport
	VolumesReport   types.VolumesPruneReport
}

//TotalSpaceReclaimed reports the total space reclaimed
func (p *PruneReport) TotalSpaceReclaimed() uint64 {
	total := p.ContainerReport.SpaceReclaimed
	total += p.ImagesReport.SpaceReclaimed
	total += p.VolumesReport.SpaceReclaimed
	return total
}
