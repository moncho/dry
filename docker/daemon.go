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
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	dockerAPI "github.com/docker/docker/client"
)

const (
	//DefaultDockerHost is used as a default docker host to connect to
	//if no other value is given.
	DefaultDockerHost = "unix:///var/run/docker.sock"
)

// timeout in seconds for docker operations
var defaultOperationTimeout = time.Duration(10) * time.Second


// Defaults for listing images
var defaultImageListOptions = image.ListOptions{
	All: false}

// DockerDaemon knows how to talk to the Docker daemon
type DockerDaemon struct {
	client    dockerAPI.APIClient //client used to to connect to the Docker daemon
	s         ContainerStore
	err       error // Errors, if any.
	dockerEnv Env
	version   *dockerTypes.Version
	swarmMode bool
	storeLock sync.RWMutex
	resolver  Resolver
	eventLog  *EventLog
}

// Containers returns the containers known by the daemon
func (daemon *DockerDaemon) Containers(filters []ContainerFilter, mode SortMode) []*Container {
	c := daemon.store().List()
	for _, filter := range filters {
		c = filter.Apply(c)
	}
	SortContainers(c, mode)
	return c
}

// ContainerByID returns the container with the given ID
func (daemon *DockerDaemon) ContainerByID(cid string) *Container {
	return daemon.store().Get(cid)
}

// DiskUsage returns reported Docker disk usage
func (daemon *DockerDaemon) DiskUsage() (dockerTypes.DiskUsage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.DiskUsage(ctx, dockerTypes.DiskUsageOptions{})
}

// DockerEnv returns Docker-related environment variables
func (daemon *DockerDaemon) DockerEnv() Env {
	return daemon.dockerEnv
}

// Events returns a channel to receive Docker events.
// The caller owns cancellation via the provided context.
// The returned channel is closed when the context is cancelled or the
// Docker daemon disconnects (error on the event stream).
func (daemon *DockerDaemon) Events(ctx context.Context) (<-chan dockerEvents.Message, error) {
	// Derive an internal context so error on either event stream
	// cancels all goroutines and closes the output channel.
	innerCtx, innerCancel := context.WithCancel(ctx)

	args := filters.NewArgs()
	args.Add("scope", "local")
	options := dockerEvents.ListOptions{
		Filters: args,
	}
	events, err := daemon.client.Events(innerCtx, options)

	eventC := make(chan dockerEvents.Message)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case event := <-events:
				if event.Action != "top" {
					handleEvent(
						innerCtx,
						event,
						streamEvents(eventC),
						logEvents(daemon.eventLog),
						callbackNotifier)
				}
			case <-err:
				innerCancel()
				return
			case <-innerCtx.Done():
				return
			}
		}
	}()

	if daemon.swarmMode {
		swarmArgs := filters.NewArgs()
		swarmArgs.Add("scope", "swarm")
		swarmOptions := dockerEvents.ListOptions{
			Filters: swarmArgs,
		}
		swarmEvents, swarmErr := daemon.client.Events(innerCtx, swarmOptions)

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case event := <-swarmEvents:
					handleEvent(
						innerCtx,
						event,
						streamEvents(eventC),
						logEvents(daemon.eventLog),
						callbackNotifier)
				case <-swarmErr:
					innerCancel()
					return
				case <-innerCtx.Done():
					return
				}
			}
		}()
	}

	// Cleanup goroutine: waits for all event goroutines to finish,
	// then closes the output channel. Goroutines exit on context
	// cancellation (caller or internal) or Docker stream error.
	go func() {
		wg.Wait()
		innerCancel() // no-op if already cancelled, prevents leak
		close(eventC)
	}()

	return eventC, nil
}

// EventLog returns the events log
func (daemon *DockerDaemon) EventLog() *EventLog {
	return daemon.eventLog
}

// Info returns system-wide information about the Docker server.
func (daemon *DockerDaemon) Info() (system.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	return daemon.client.Info(ctx)
}

// Inspect the container with the given id
func (daemon *DockerDaemon) Inspect(id string) (container.InspectResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	return daemon.client.ContainerInspect(ctx, id)
}

// InspectImage the image with the name
func (daemon *DockerDaemon) InspectImage(name string) (image.InspectResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	inspect, _, err := daemon.client.ImageInspectWithRaw(ctx, name)
	return inspect, err
}

// IsContainerRunning returns true if the container with the given  is running
func (daemon *DockerDaemon) IsContainerRunning(id string) bool {
	return IsContainerRunning(daemon.store().Get(id))
}

// Kill the container with the given id
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

// Logs shows the logs of the container with the given id
func (daemon *DockerDaemon) Logs(id string, since string, withTimeStamps bool) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: withTimeStamps,
		Follow:     true,
		Details:    false,
		Since:      since,
	}
	return daemon.client.ContainerLogs(context.Background(), id, options)
}

// Networks returns the list of Docker networks
func (daemon *DockerDaemon) Networks() ([]network.Inspect, error) {
	return networks(daemon.client)
}

// NetworkInspect returns network detailed information
func (daemon *DockerDaemon) NetworkInspect(id string) (network.Inspect, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	options := network.InspectOptions{
		Verbose: true,
	}
	return daemon.client.NetworkInspect(
		ctx, id, options)
}

// Ok is true if connecting to the Docker daemon went fine
func (daemon *DockerDaemon) Ok() (bool, error) {
	return daemon.err == nil, daemon.err
}

// StatsChannel creates a channel with the runtime stats of the given container
func (daemon *DockerDaemon) StatsChannel(container *Container) (*StatsChannel, error) {
	return newStatsChannel(daemon.version, daemon.client, container)
}

// Prune requests the Docker daemon to prune unused containers, images
// networks and volumes
func (daemon *DockerDaemon) Prune() (*PruneReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	args := filters.NewArgs()
	cReport, err := daemon.client.ContainersPrune(ctx, args)
	if err != nil {
		return nil, err
	}
	iReport, err := daemon.client.ImagesPrune(ctx, args)
	if err != nil {
		return nil, err
	}
	nReport, err := daemon.client.NetworksPrune(ctx, args)
	if err != nil {
		return nil, err
	}
	vRreport, err := daemon.client.VolumesPrune(ctx, args)
	if err != nil {
		return nil, err
	}
	return &PruneReport{
		ContainerReport: cReport,
		ImagesReport:    iReport,
		NetworksReport:  nReport,
		VolumesReport:   vRreport}, nil
}

// RestartContainer restarts the container with the given id
func (daemon *DockerDaemon) RestartContainer(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	// Default timeout is 10 seconds
	if err := daemon.client.ContainerRestart(ctx, id, container.StopOptions{}); err != nil {
		return err
	}

	return daemon.refreshAndWait()
}

// Refresh the container list asynchronously, using the given notifier to signal
// operation completion.
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

// RemoveAllStoppedContainers removes all stopped containers
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
			fmt.Errorf("remove stopped containers, containers: %d, removed: %d: %w", len(containers), removed, e)

	default:
	}
	err := daemon.refreshAndWait()
	return removed, err
}

// RemoveDanglingImages removes dangling images
func (daemon *DockerDaemon) RemoveDanglingImages() (int, error) {
	danglingfilters := filters.NewArgs()
	danglingfilters.Add("dangling", "true")
	images, err := images(daemon.client,
		image.ListOptions{
			Filters: danglingfilters})
	var count uint32
	errs := make(chan error, 1)
	defer close(errs)
	if err == nil {
		var wg sync.WaitGroup
		for _, image := range images {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				_, localErr := daemon.Rmi(id, true)
				if localErr != nil {
					select {
					case errs <- localErr:
					default:
					}
				} else {
					atomic.AddUint32(&count, 1)
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

// RemoveUnusedImages removes unused images
func (daemon *DockerDaemon) RemoveUnusedImages() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	args := filters.NewArgs()
	args.Add("dangling", "false")

	report, err := daemon.client.ImagesPrune(ctx, args)

	return len(report.ImagesDeleted), err
}

// RemoveNetwork removes the network with the given id
func (daemon *DockerDaemon) RemoveNetwork(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.NetworkRemove(ctx, id)
}

// Rm removes the container with the given id
func (daemon *DockerDaemon) Rm(id string) error {

	opts := container.RemoveOptions{
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

// Rmi removes the image with the given name
func (daemon *DockerDaemon) Rmi(name string, force bool) ([]image.DeleteResponse, error) {
	options := image.RemoveOptions{
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

// StopContainer stops the container with the given id
func (daemon *DockerDaemon) StopContainer(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	err := daemon.client.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
		return err
	}

	return daemon.refreshAndWait()
}

// Top returns Top information for the given container
func (daemon *DockerDaemon) Top(ctx context.Context, id string) (container.TopResponse, error) {
	return daemon.client.ContainerTop(ctx, id, nil)
}

// VolumeInspect returns the details of the given volume.
func (daemon *DockerDaemon) VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error) {
	return daemon.client.VolumeInspect(ctx, volumeID)
}

// VolumeList returns the list of volumes.
func (daemon *DockerDaemon) VolumeList(ctx context.Context) ([]*volume.Volume, error) {
	volumeOkBody, err := daemon.client.VolumeList(ctx, volume.ListOptions{})

	if err != nil {
		return nil, err
	}
	return volumeOkBody.Volumes, nil
}

// VolumePrune removes unused volumes.
func (daemon *DockerDaemon) VolumePrune(ctx context.Context) (int, error) {
	pruneReport, err := daemon.client.VolumesPrune(ctx, filters.Args{})

	return len(pruneReport.VolumesDeleted), err
}

// VolumeRemove removes the given volume.
func (daemon *DockerDaemon) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return daemon.client.VolumeRemove(ctx, volumeID, force)
}

// VolumeRemoveAll removes all the volumes.
func (daemon *DockerDaemon) VolumeRemoveAll(ctx context.Context) (int, error) {
	volumes, err := daemon.VolumeList(ctx)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, vol := range volumes {
		err := daemon.VolumeRemove(ctx, vol.Name, true)
		if err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// Version returns version information about the Docker Engine
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

// init initializes the internals of the docker daemon.
func (daemon *DockerDaemon) init() error {
	daemon.eventLog = NewEventLog()
	//This loads Docker Version information
	if _, err := daemon.Version(); err != nil {
		return fmt.Errorf("get Docker version: %w", err)
	}

	if info, err := daemon.Info(); err == nil {
		daemon.swarmMode = info.Swarm.LocalNodeState == swarm.LocalNodeStateActive
	} else {
		return fmt.Errorf("get Docker info: %w", err)
	}
	GlobalRegistry.Register(
		ContainerSource,
		func(ctx context.Context, message dockerEvents.Message) {
			_ = daemon.refreshAndWait()
		})
	return nil
}

func containers(client dockerAPI.ContainerAPIClient) ([]*Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectionTimeout)
	defer cancel()
	containers, err := client.ContainerList(ctx, container.ListOptions{All: true, Size: true})
	if err != nil {
		return nil, fmt.Errorf("retrieve container list: %w", err)
	}

	var cc []*Container
	for i, c := range containers {
		details, err := client.ContainerInspect(ctx, c.ID)
		if err != nil {
			return nil, fmt.Errorf("inspect container %s: %w", c.ID, err)
		}
		cc = append(cc, &Container{Summary: containers[i], Detail: details})
	}
	return cc, nil

}

func images(client dockerAPI.ImageAPIClient, opts image.ListOptions) ([]image.Summary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return client.ImageList(ctx, opts)
}

func networks(client dockerAPI.NetworkAPIClient) ([]network.Inspect, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	networks, err := client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, err
	}

	detailedNetworks := make([]network.Inspect, len(networks))
	options := network.InspectOptions{
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

// GetBool returns false if the given string looks like you mean
// false, true otherwise. Func does not belong here.
func GetBool(key string) (value bool) {
	s := strings.ToLower(strings.Trim(key, " "))
	if s == "" || s == "0" || s == "no" || s == "false" || s == "none" {
		return false
	}
	return true
}

// IsContainerRunning returns true if the given container is running
func IsContainerRunning(container *Container) bool {
	if container != nil {
		return strings.Contains(container.Status, "Up")
	}
	return false
}
