package docker

import (
	"io"

	"github.com/fsouza/go-dockerclient"
	godocker "github.com/fsouza/go-dockerclient"
)

//ContainerDaemon describes what is expected from the container daemon
type ContainerDaemon interface {
	Containers() []docker.APIContainers
	ContainersCount() int
	ContainerIDAt(pos int) (string, string, error)
	ContainerByID(cid string) docker.APIContainers
	DockerEnv() *DockerEnv
	Events() (chan *docker.APIEvents, error)
	Info() (*docker.Env, error)
	Inspect(id string) (*docker.Container, error)
	IsContainerRunning(id string) bool
	Kill(id string) error
	Logs(id string) io.ReadCloser
	Ok() (bool, error)
	RestartContainer(id string) error
	Rm(id string) error
	Refresh(allContainers bool) error
	RemoveAllStoppedContainers() error
	Stats(id string) (<-chan *Stats, chan<- bool, <-chan error)
	StopContainer(id string) error
	Sort(sortMode SortMode)
	StopReceivingEvents(eventChan chan *docker.APIEvents) error
	Version() (*Version, error)
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
	Stats            *godocker.Stats
}

// Version contains response of Remote API:
// GET "/version"
//Copied from docker/engine-api/types until docker library is fully replaced
type Version struct {
	Version       string
	APIVersion    string
	GitCommit     string
	GoVersion     string
	Os            string
	Arch          string
	KernelVersion string
	Experimental  bool
	BuildTime     string
}
