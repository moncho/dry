package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerEvents "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	dockerAPI "github.com/docker/docker/client"
	pkgError "github.com/pkg/errors"
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
	client    dockerAPI.APIClient //client used to to connect to the Docker daemon
	s         ContainerStore
	err       error // Errors, if any.
	dockerEnv *Env
	version   *dockerTypes.Version
	swarmMode bool
	storeLock sync.RWMutex
	resolver  Resolver
	eventLog  *EventLog
}

//Containers returns the containers known by the daemon
func (daemon *DockerDaemon) Containers(filters []ContainerFilter, mode SortMode) []*Container {
	c := daemon.store().List()
	for _, filter := range filters {
		c = filter.Apply(c)
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.DiskUsage(ctx)
}

//DockerEnv returns Docker-related environment variables
func (daemon *DockerDaemon) DockerEnv() *Env {
	return daemon.dockerEnv
}

// Events returns a channel to receive Docker events.
func (daemon *DockerDaemon) Events() (<-chan dockerEvents.Message, chan<- struct{}, error) {

	args := filters.NewArgs()

	args.Add("scope", "local")

	options := dockerTypes.EventsOptions{
		Filters: args,
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
						logEvents(daemon.eventLog),
						callbackNotifier); err != nil {
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

	if daemon.swarmMode {
		args := filters.NewArgs()
		args.Add("scope", "swarm")
		options := dockerTypes.EventsOptions{
			Filters: args,
		}

		swarmEvents, err := daemon.client.Events(ctx, options)

		go func() {

			for {
				select {
				case event := <-swarmEvents:
					if err := handleEvent(
						ctx,
						event,
						streamEvents(eventC),
						logEvents(daemon.eventLog),
						callbackNotifier); err != nil {
						return
					}
				case <-err:
					return
				case <-done:
					return
				}
			}

		}()
	}

	return eventC, done, nil
}

//EventLog returns the events log
func (daemon *DockerDaemon) EventLog() *EventLog {
	return daemon.eventLog
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	inspect, _, err := daemon.client.ImageInspectWithRaw(ctx, name)
	return inspect, err
}

//IsContainerRunning returns true if the container with the given  is running
func (daemon *DockerDaemon) IsContainerRunning(id string) bool {
	return IsContainerRunning(daemon.store().Get(id))
}

//Kill the container with the given id
func (daemon *DockerDaemon) Kill(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	//TODO Send signal?
	err := daemon.client.ContainerKill(ctx, id, "")
	if err != nil {
		return err
	}
	return daemon.refreshAndWait()
}

//Logs shows the logs of the container with the given id
func (daemon *DockerDaemon) Logs(id string, since string) (io.ReadCloser, error) {
	options := dockerTypes.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Follow:     true,
		Details:    false,
		Since:      since,
	}
	return daemon.client.ContainerLogs(context.Background(), id, options)
}

//Networks returns the list of Docker networks
func (daemon *DockerDaemon) Networks() ([]dockerTypes.NetworkResource, error) {
	return networks(daemon.client)
}

//NetworkInspect returns network detailed information
func (daemon *DockerDaemon) NetworkInspect(id string) (dockerTypes.NetworkResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	options := dockerTypes.NetworkInspectOptions{
		Verbose: true,
	}
	return daemon.client.NetworkInspect(
		ctx, id, options)
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

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

//RemoveAllStoppedContainers removes all stopped containers
func (daemon *DockerDaemon) RemoveAllStoppedContainers() (int, error) {
	containers := daemon.Containers([]ContainerFilter{ContainerFilters.NotRunning()}, NoSort)
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.NetworkRemove(ctx, id)
}

//Rm removes the container with the given id
func (daemon *DockerDaemon) Rm(id string) error {

	opts := dockerTypes.ContainerRemoveOptions{
		RemoveVolumes: false,
		RemoveLinks:   false,
		Force:         true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	err := daemon.client.ContainerRemove(ctx, id, opts)
	if err == nil {
		daemon.store().Remove(id)
	}
	return err
}

//Rmi removes the image with the given name
func (daemon *DockerDaemon) Rmi(name string, force bool) ([]dockerTypes.ImageDeleteResponseItem, error) {
	options := dockerTypes.ImageRemoveOptions{
		Force: force,
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	err := daemon.client.ContainerStop(ctx, id, &containerOpTimeout)
	if err != nil {
		return err
	}

	return daemon.refreshAndWait()
}

//Top returns Top information for the given container
func (daemon *DockerDaemon) Top(id string) (container.ContainerTopOKBody, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.ContainerTop(ctx, id, nil)
}

//Version returns version information about the Docker Engine
func (daemon *DockerDaemon) Version() (*dockerTypes.Version, error) {
	if daemon.version == nil {
		ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
		defer cancel()
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
	GlobalRegistry.Register(
		ContainerSource,
		func(ctx context.Context, message dockerEvents.Message) error {
			return daemon.refreshAndWait()
		})
}

func containers(client dockerAPI.ContainerAPIClient) ([]*Container, error) {
	//Since this is how dry fist connects to the Docker daemon
	//a different (longer) timeout is used.
	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectionTimeout)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return client.ImageList(ctx, opts)
}

func networks(client dockerAPI.NetworkAPIClient) ([]dockerTypes.NetworkResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	networks, err := client.NetworkList(ctx, dockerTypes.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	detailedNetworks := make([]dockerTypes.NetworkResource, len(networks))
	options := dockerTypes.NetworkInspectOptions{
		Verbose: true,
	}
	for i, n := range networks {
		detailedNetwork, err := client.NetworkInspect(ctx, n.ID, options)
		if err != nil {
			return nil, err
		}

		detailedNetworks[i] = detailedNetwork

	}

	return detailedNetworks, nil
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
