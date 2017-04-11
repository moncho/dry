package docker

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	dockerEvents "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	dockerAPI "github.com/docker/docker/client"
	pkgError "github.com/pkg/errors"
	"golang.org/x/net/context"
)

const (
	//DefaultDockerHost is used as a default docker host to connect to
	//if no other value is given.
	DefaultDockerHost = "unix:///var/run/docker.sock"
)

//timeout in seconds for docker operations
var defaultOperationTimeout = time.Duration(10) * time.Second

//container operations timeout
var containerOpTimeout = time.Duration(10) * time.Second

//Defaults for listing images
var defaultImageListOptions = dockerTypes.ImageListOptions{
	All: false}

//DockerDaemon knows how to talk to the Docker daemon
type DockerDaemon struct {
	client       dockerAPI.APIClient //client used to to connect to the Docker daemon
	s            ContainerStore
	images       []dockerTypes.ImageSummary
	networks     []dockerTypes.NetworkResource
	err          error // Errors, if any.
	connected    bool
	dockerEnv    *Env
	version      *dockerTypes.Version
	swarmMode    bool
	storeLock    sync.RWMutex
	imagesLock   sync.RWMutex
	networksLock sync.RWMutex

	eventLog *EventLog
}

func init() {
	log.SetOutput(os.Stderr)
}

//Containers returns the containers known by the daemon
func (daemon *DockerDaemon) Containers(filter ContainerFilter, mode SortMode) []*Container {
	c := daemon.store().List()
	if filter != nil {
		c = Filter(c, filter)
	}
	SortContainers(c, mode)
	return c
}

//ContainerCount returns the total number of containers.
func (daemon *DockerDaemon) ContainerCount() int {
	return daemon.store().Size()
}

//ContainerByID returns the container with the given ID
func (daemon *DockerDaemon) ContainerByID(cid string) *Container {
	return daemon.store().Get(cid)
}

//DiskUsage returns reported Docker disk usage
func (daemon *DockerDaemon) DiskUsage() (dockerTypes.DiskUsage, error) {
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)
	return daemon.client.DiskUsage(ctx)
}

//DockerEnv returns Docker-related environment variables
func (daemon *DockerDaemon) DockerEnv() *Env {
	return daemon.dockerEnv
}

// Events returns a channel to receive Docker events.
func (daemon *DockerDaemon) Events() (<-chan dockerEvents.Message, chan<- struct{}, error) {

	options := dockerTypes.EventsOptions{
	//Since: time.Now().String(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	events, err := daemon.client.Events(ctx, options)

	eventC := make(chan dockerEvents.Message)
	done := make(chan struct{})

	go func() {
		defer cancel()
		defer close(eventC)
		for {
			select {
			case event := <-events:
				if event.Action != "top" {
					if err := handleEvent(
						ctx,
						event,
						streamEvents(eventC),
						logEvents(daemon.eventLog)); err != nil {
						return
					}
				}
			case <-err:
				return
			case <-done:
				return
			}
		}

	}()

	return eventC, done, nil
}

//EventLog returns the events log
func (daemon *DockerDaemon) EventLog() *EventLog {
	return daemon.eventLog
}

//History returns image history
func (daemon *DockerDaemon) History(id string) ([]dockerTypes.ImageHistory, error) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	return daemon.client.ImageHistory(
		ctx, id)
}

//ImageAt returns the Image found at the given
//position.
func (daemon *DockerDaemon) ImageAt(pos int) (*dockerTypes.ImageSummary, error) {
	daemon.imagesLock.Lock()
	defer daemon.imagesLock.Unlock()
	if pos >= len(daemon.images) {
		return nil, errors.New("Position is higher than number of images")
	}
	return &daemon.images[pos], nil
}

//Images returns the list of Docker images
func (daemon *DockerDaemon) Images() ([]dockerTypes.ImageSummary, error) {
	daemon.imagesLock.Lock()
	defer daemon.imagesLock.Unlock()
	return daemon.images, nil
}

//ImagesCount returns the number of images
func (daemon *DockerDaemon) ImagesCount() int {
	daemon.imagesLock.Lock()
	defer daemon.imagesLock.Unlock()
	return len(daemon.images)
}

//Info returns system-wide information about the Docker server.
func (daemon *DockerDaemon) Info() (dockerTypes.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	return daemon.client.Info(ctx)
}

//Inspect the container with the given id
func (daemon *DockerDaemon) Inspect(id string) (dockerTypes.ContainerJSON, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	return daemon.client.ContainerInspect(ctx, id)
}

//InspectImage the image with the name
func (daemon *DockerDaemon) InspectImage(name string) (dockerTypes.ImageInspect, error) {
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)

	inspect, _, err := daemon.client.ImageInspectWithRaw(ctx, name)
	return inspect, err
}

//IsContainerRunning returns true if the container with the given  is running
func (daemon *DockerDaemon) IsContainerRunning(id string) bool {
	return IsContainerRunning(daemon.store().Get(id))
}

//Kill the container with the given id
func (daemon *DockerDaemon) Kill(id string) error {
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)
	//TODO Send signal?
	err := daemon.client.ContainerKill(ctx, id, "")
	if err != nil {
		return err
	}
	return daemon.refreshAndWait()
}

//Logs shows the logs of the container with the given id
func (daemon *DockerDaemon) Logs(id string) io.ReadCloser {
	options := dockerTypes.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Follow:     true,
		Details:    false,
	}
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)

	reader, _ := daemon.client.ContainerLogs(ctx, id, options)
	return reader
}

//Networks returns the list of Docker networks
func (daemon *DockerDaemon) Networks() ([]dockerTypes.NetworkResource, error) {
	daemon.networksLock.RLock()
	defer daemon.networksLock.RUnlock()
	return daemon.networks, nil
}

//NetworkAt returns the network found at the given position.
func (daemon *DockerDaemon) NetworkAt(pos int) (*dockerTypes.NetworkResource, error) {
	daemon.networksLock.RLock()
	defer daemon.networksLock.RUnlock()
	if pos >= len(daemon.networks) {
		return nil, errors.New("Position is higher than number of networks")
	}
	return &daemon.networks[pos], nil
}

//NetworksCount returns the number of networks reported by Docker
func (daemon *DockerDaemon) NetworksCount() int {
	daemon.networksLock.RLock()
	defer daemon.networksLock.RUnlock()
	return len(daemon.networks)
}

//NetworkInspect returns network detailed information
func (daemon *DockerDaemon) NetworkInspect(id string) (dockerTypes.NetworkResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	return daemon.client.NetworkInspect(
		ctx, id)
}

//Ok is true if connecting to the Docker daemon went fine
func (daemon *DockerDaemon) Ok() (bool, error) {
	return daemon.err == nil, daemon.err
}

//OpenChannel creates a channel with the runtime stats of the given container
func (daemon *DockerDaemon) OpenChannel(container *Container) *StatsChannel {
	return NewStatsChannel(daemon, container)
}

//Prune requests the Docker daemon to prune unused containers, images
//networks and volumes
func (daemon *DockerDaemon) Prune() (*PruneReport, error) {
	c := context.Background()

	args := filters.NewArgs()
	args.Add("force", "y")
	cReport, err := daemon.client.ContainersPrune(c, args)
	if err != nil {
		return nil, err
	}
	iReport, err := daemon.client.ImagesPrune(c, args)
	if err != nil {
		return nil, err
	}
	nReport, err := daemon.client.NetworksPrune(c, args)
	if err != nil {
		return nil, err
	}
	vRreport, err := daemon.client.VolumesPrune(c, args)
	if err != nil {
		return nil, err
	}
	return &PruneReport{
		ContainerReport: cReport,
		ImagesReport:    iReport,
		NetworksReport:  nReport,
		VolumesReport:   vRreport}, nil

}

//RestartContainer restarts the container with the given id
func (daemon *DockerDaemon) RestartContainer(id string) error {
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)

	//fixme: timeout to start a container
	if err := daemon.client.ContainerRestart(ctx, id, &containerOpTimeout); err != nil {
		return err
	}

	return daemon.refreshAndWait()
}

//Refresh the container list asynchronously, using the given notifier to signal
//operation completion.
func (daemon *DockerDaemon) Refresh(notify func(error)) {
	go func() {
		store, err := NewDockerContainerStore(daemon.client)
		if err == nil {
			daemon.setStore(store)
		}
		if notify != nil {
			notify(err)
		}
	}()
}

func (daemon *DockerDaemon) refreshAndWait() error {
	var wg sync.WaitGroup
	var refreshError error
	wg.Add(1)
	daemon.Refresh(func(err error) {
		refreshError = err
		wg.Done()
	})
	wg.Wait()
	return refreshError
}

//RefreshImages refreshes the image list
func (daemon *DockerDaemon) RefreshImages() error {
	daemon.imagesLock.Lock()
	defer daemon.imagesLock.Unlock()

	images, err := images(daemon.client, defaultImageListOptions)

	if err == nil {
		daemon.images = images
	}
	return err
}

//RefreshNetworks refreshes the network list
func (daemon *DockerDaemon) RefreshNetworks() error {
	daemon.networksLock.Lock()
	defer daemon.networksLock.Unlock()

	networks, err := networks(daemon.client)

	if err == nil {
		daemon.networks = networks
	}
	return err
}

//RemoveAllStoppedContainers removes all stopped containers
func (daemon *DockerDaemon) RemoveAllStoppedContainers() (int, error) {
	containers := daemon.Containers(ContainerFilters.NotRunning(), NoSort)
	var count uint32
	errs := make(chan error, 1)
	defer close(errs)
	var wg sync.WaitGroup
	for _, container := range containers {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			err := daemon.Rm(id)
			if err != nil {
				select {
				case errs <- err:
				default:
				}
			} else {
				atomic.AddUint32(&count, 1)
			}
		}(container.ID)
	}

	wg.Wait()
	removed := int(atomic.LoadUint32(&count))
	select {
	case e := <-errs:
		return removed,
			pkgError.Wrap(e,
				fmt.Sprintf("There were errors removing stopped containers. Containers: %d, removed: %d", len(containers), removed))
	default:
	}
	err := daemon.refreshAndWait()
	return removed, err
}

//RemoveDanglingImages removes dangling images
func (daemon *DockerDaemon) RemoveDanglingImages() (int, error) {
	danglingfilters := filters.NewArgs()
	danglingfilters.Add("dangling", "true")
	images, err := images(daemon.client,
		dockerTypes.ImageListOptions{
			Filters: danglingfilters})
	var count uint32
	errs := make(chan error, 1)
	defer close(errs)
	if err == nil {
		var wg sync.WaitGroup
		for _, image := range images {
			wg.Add(1)
			go func(id string) {
				defer atomic.AddUint32(&count, 1)
				defer wg.Done()
				_, err = daemon.Rmi(id, true)
				if err != nil {
					select {
					case errs <- err:
					default:
					}
				}
			}(image.ID)
		}
		wg.Wait()
		select {
		case e := <-errs:
			return 0, e
		default:
		}
	}
	daemon.Refresh(nil)
	return int(atomic.LoadUint32(&count)), err
}

//RemoveNetwork removes the network with the given id
func (daemon *DockerDaemon) RemoveNetwork(id string) error {
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)

	return daemon.client.NetworkRemove(ctx, id)
}

//Rm removes the container with the given id
func (daemon *DockerDaemon) Rm(id string) error {

	opts := dockerTypes.ContainerRemoveOptions{
		RemoveVolumes: false,
		RemoveLinks:   false,
		Force:         true,
	}
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)
	err := daemon.client.ContainerRemove(ctx, id, opts)
	if err == nil {
		daemon.store().Remove(id)
	}
	return err
}

//Rmi removes the image with the given name
func (daemon *DockerDaemon) Rmi(name string, force bool) ([]dockerTypes.ImageDelete, error) {
	options := dockerTypes.ImageRemoveOptions{
		Force: force,
	}
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)

	return daemon.client.ImageRemove(ctx, name, options)
}

func (daemon *DockerDaemon) setStore(store ContainerStore) {
	daemon.storeLock.Lock()
	defer daemon.storeLock.Unlock()
	daemon.s = store
}

func (daemon *DockerDaemon) store() ContainerStore {
	daemon.storeLock.RLock()
	defer daemon.storeLock.RUnlock()
	return daemon.s
}

//Stats shows resource usage statistics of the container with the given id
func (daemon *DockerDaemon) Stats(id string) (<-chan *Stats, chan<- struct{}) {
	stream := NewStatsChannel(daemon, daemon.store().Get(id))
	return stream.Stats, stream.Done
}

//StopContainer stops the container with the given id
func (daemon *DockerDaemon) StopContainer(id string) error {
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)

	err := daemon.client.ContainerStop(ctx, id, &containerOpTimeout)
	if err != nil {
		return err
	}

	return daemon.refreshAndWait()
}

//SortImages sorts the list of images by the given mode
func (daemon *DockerDaemon) SortImages(sortMode SortImagesMode) {
	daemon.imagesLock.Lock()
	defer daemon.imagesLock.Unlock()
	SortImages(daemon.images, sortMode)
}

//SortNetworks sortes the list of networks by the given mode
func (daemon *DockerDaemon) SortNetworks(sortMode SortNetworksMode) {
	daemon.networksLock.Lock()
	defer daemon.networksLock.Unlock()
	SortNetworks(daemon.networks, sortMode)
}

//Top returns Top information for the given container
func (daemon *DockerDaemon) Top(id string) (dockerTypes.ContainerProcessList, error) {
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)

	return daemon.client.ContainerTop(ctx, id, nil)
}

//Version returns version information about the Docker Engine
func (daemon *DockerDaemon) Version() (*dockerTypes.Version, error) {
	if daemon.version == nil {
		//TODO use cancel function
		ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)

		v, err := daemon.client.ServerVersion(ctx)
		if err == nil {
			daemon.version = &v
			return daemon.version, nil
		}
		return nil, err
	}
	return daemon.version, nil
}

//init initializes the internals of the docker daemon.
func (daemon *DockerDaemon) init() {
	daemon.eventLog = NewEventLog()
	daemon.Version()
	if info, err := daemon.Info(); err == nil {
		daemon.swarmMode = info.Swarm.LocalNodeState == swarm.LocalNodeStateActive
	}
}

func containers(client dockerAPI.ContainerAPIClient) ([]*Container, error) {
	//TODO use cancel function
	//Since this is how dry fist connects to the Docker daemon
	//a different (longer) timeout is used.
	ctx, _ := context.WithTimeout(context.Background(), DefaultConnectionTimeout)

	containers, err := client.ContainerList(ctx, dockerTypes.ContainerListOptions{All: true, Size: true})
	if err == nil {
		var cPointers []*Container
		for i, c := range containers {
			details, _ := client.ContainerInspect(ctx, c.ID)
			cPointers = append(cPointers, &Container{containers[i], details})
		}
		return cPointers, nil
	}
	return nil, pkgError.Wrap(err, "Error retrieving container list")
}

func images(client dockerAPI.ImageAPIClient, opts dockerTypes.ImageListOptions) ([]dockerTypes.ImageSummary, error) {
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)

	return client.ImageList(ctx, opts)
}

func networks(client dockerAPI.NetworkAPIClient) ([]dockerTypes.NetworkResource, error) {
	//TODO use cancel function
	ctx, _ := context.WithTimeout(context.Background(), defaultOperationTimeout)

	return client.NetworkList(ctx, dockerTypes.NetworkListOptions{})
}

//GetBool returns false if the given string looks like you mean
//false, true otherwise. Func does not belong here.
func GetBool(key string) (value bool) {
	s := strings.ToLower(strings.Trim(key, " "))
	if s == "" || s == "0" || s == "no" || s == "false" || s == "none" {
		return false
	}
	return true
}

//IsContainerRunning returns true if the given container is running
func IsContainerRunning(container *Container) bool {
	if container != nil {
		return strings.Contains(container.Status, "Up")
	}
	return false
}
