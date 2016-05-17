package docker

import (
	"io"

	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/events"
)

//ContainerDaemon describes what is expected from the container daemon
type ContainerDaemon interface {
	Containers() []types.Container
	ContainerAt(pos int) (types.Container, error)
	ContainersCount() int
	ContainerIDAt(pos int) (string, string, error)
	ContainerByID(cid string) types.Container
	DockerEnv() *DockerEnv
	Events() (<-chan events.Message, chan<- struct{}, error)
	EventLog() *EventLog
	History(id string) ([]types.ImageHistory, error)
	ImageAt(pos int) (*types.Image, error)
	Images() ([]types.Image, error)
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
	RestartContainer(id string) error
	Rm(id string) error
	Rmi(id string, force bool) ([]types.ImageDelete, error)
	Refresh(allContainers bool) error
	RefreshImages() error
	RefreshNetworks() error
	RemoveAllStoppedContainers() (int, error)
	Stats(id string) (<-chan *Stats, chan<- struct{})
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
