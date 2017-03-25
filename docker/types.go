package docker

import (
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
)

//ContainerDaemon describes what is expected from the container daemon
type ContainerDaemon interface {
	ContainerStore() *ContainerStore
	DiskUsage() (types.DiskUsage, error)
	DockerEnv() *Env
	Events() (<-chan events.Message, chan<- struct{}, error)
	EventLog() *EventLog
	History(id string) ([]types.ImageHistory, error)
	ImageAt(pos int) (*types.ImageSummary, error)
	Images() ([]types.ImageSummary, error)
	ImagesCount() int
	Info() (types.Info, error)
	Inspect(id string) (types.ContainerJSON, error)
	InspectImage(id string) (types.ImageInspect, error)
	IsContainerRunning(id string) bool
	Kill(id string) error
	Logs(id string) io.ReadCloser
	Networks() ([]types.NetworkResource, error)
	NetworkAt(pos int) (*types.NetworkResource, error)
	NetworksCount() int
	NetworkInspect(id string) (types.NetworkResource, error)
	Ok() (bool, error)
	OpenChannel(container *types.Container) *StatsChannel
	Prune() (*PruneReport, error)
	RestartContainer(id string) error
	Rm(id string) error
	Rmi(id string, force bool) ([]types.ImageDelete, error)
	Refresh(allContainers bool) error
	RefreshImages() error
	RefreshNetworks() error
	RemoveAllStoppedContainers() (int, error)
	RemoveDanglingImages() (int, error)
	RemoveNetwork(id string) error
	StopContainer(id string) error
	Sort(sortMode SortMode)
	SortImages(sortMode SortImagesMode)
	SortNetworks(sortMode SortNetworksMode)
	Top(id string) (types.ContainerProcessList, error)
	Version() (*types.Version, error)
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
