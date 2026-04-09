package docker

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
)

const (
	// DefaultDockerHost is used as a default docker host to connect to
	// if no other value is given.
	DefaultDockerHost = "unix:///var/run/docker.sock"
)

// timeout in seconds for docker operations
var defaultOperationTimeout = time.Duration(10) * time.Second

// Defaults for listing images
var defaultImageListOptions = client.ImageListOptions{
	All: false,
}

// DockerDaemon knows how to talk to the Docker daemon
type DockerDaemon struct {
	client    client.APIClient // client used to connect to the Docker daemon
	s         ContainerStore
	err       error // Errors, if any.
	dockerEnv Env
	version   *client.ServerVersionResult
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
func (daemon *DockerDaemon) DiskUsage() (client.DiskUsageResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return daemon.client.DiskUsage(ctx, client.DiskUsageOptions{})
}

// DockerEnv returns Docker-related environment variables
func (daemon *DockerDaemon) DockerEnv() Env {
	return daemon.dockerEnv
}

// Events returns a channel to receive Docker events.
// The caller owns cancellation via the provided context.
// The returned channel is closed when the context is cancelled or the
// Docker daemon disconnects (error on the event stream).
func (daemon *DockerDaemon) Events(ctx context.Context) (<-chan events.Message, error) {
	// Derive an internal context so error on either event stream
	// cancels all goroutines and closes the output channel.
	innerCtx, innerCancel := context.WithCancel(ctx)

	res := daemon.client.Events(innerCtx, client.EventsListOptions{
		Filters: make(client.Filters).Add("scope", "local"),
	})

	eventC := make(chan events.Message)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case event := <-res.Messages:
				if event.Action != "top" {
					handleEvent(
						innerCtx,
						event,
						streamEvents(eventC),
						logEvents(daemon.eventLog),
						callbackNotifier)
				}
			case <-res.Err:
				innerCancel()
				return
			case <-innerCtx.Done():
				return
			}
		}
	}()

	if daemon.swarmMode {
		swarmEvents := daemon.client.Events(innerCtx, client.EventsListOptions{
			Filters: make(client.Filters).Add("scope", "swarm"),
		})

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case event := <-swarmEvents.Messages:
					handleEvent(
						innerCtx,
						event,
						streamEvents(eventC),
						logEvents(daemon.eventLog),
						callbackNotifier)
				case <-swarmEvents.Err:
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

	res, err := daemon.client.Info(ctx, client.InfoOptions{})
	if err != nil {
		return system.Info{}, err
	}
	return res.Info, nil
}

// Inspect the container with the given id
func (daemon *DockerDaemon) Inspect(id string) (container.InspectResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	res, err := daemon.client.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, err
	}
	return res.Container, nil
}

// InspectImage the image with the name
func (daemon *DockerDaemon) InspectImage(name string) (image.InspectResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.ImageInspect(ctx, name)
	if err != nil {
		return image.InspectResponse{}, err
	}
	return res.InspectResponse, err
}

// IsContainerRunning returns true if the container with the given  is running
func (daemon *DockerDaemon) IsContainerRunning(id string) bool {
	return IsContainerRunning(daemon.store().Get(id))
}

// Kill the container with the given id
func (daemon *DockerDaemon) Kill(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	_, err := daemon.client.ContainerKill(ctx, id, client.ContainerKillOptions{})
	if err != nil {
		return err
	}
	return daemon.refreshAndWait()
}

// Logs shows the logs of the container with the given id
func (daemon *DockerDaemon) Logs(id string, since string, withTimeStamps bool) (io.ReadCloser, error) {
	options := client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: withTimeStamps,
		Follow:     true,
		Details:    false,
		Since:      since,
	}
	if strings.HasPrefix(since, "tail:") {
		options.Since = ""
		options.Tail = strings.TrimPrefix(since, "tail:")
		if _, err := strconv.Atoi(options.Tail); err != nil {
			return nil, fmt.Errorf("invalid log tail value %q: %w", options.Tail, err)
		}
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

	res, err := daemon.client.NetworkInspect(ctx, id, client.NetworkInspectOptions{
		Verbose: true,
	})
	if err != nil {
		return network.Inspect{}, err
	}
	return res.Network, nil
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
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	cReport, err := daemon.client.ContainerPrune(ctx, client.ContainerPruneOptions{})
	if err != nil {
		return nil, err
	}
	iReport, err := daemon.client.ImagePrune(ctx, client.ImagePruneOptions{})
	if err != nil {
		return nil, err
	}
	nReport, err := daemon.client.NetworkPrune(ctx, client.NetworkPruneOptions{})
	if err != nil {
		return nil, err
	}
	vReport, err := daemon.client.VolumePrune(ctx, client.VolumePruneOptions{})
	if err != nil {
		return nil, err
	}
	return &PruneReport{
		ContainerReport: cReport.Report,
		ImagesReport:    iReport.Report,
		NetworksReport:  nReport.Report,
		VolumesReport:   vReport.Report,
	}, nil
}

// RestartContainer restarts the container with the given id
func (daemon *DockerDaemon) RestartContainer(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()

	// Default timeout is 10 seconds
	if _, err := daemon.client.ContainerRestart(ctx, id, client.ContainerRestartOptions{}); err != nil {
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
	res, err := images(daemon.client, client.ImageListOptions{
		Filters: make(client.Filters).Add("dangling", "true"),
	})
	var count uint32
	errs := make(chan error, 1)
	defer close(errs)
	if err == nil {
		var wg sync.WaitGroup
		for _, image := range res.Items {
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

	res, err := daemon.client.ImagePrune(ctx, client.ImagePruneOptions{
		Filters: make(client.Filters).Add("dangling", "false"),
	})

	return len(res.Report.ImagesDeleted), err
}

// RemoveNetwork removes the network with the given id
func (daemon *DockerDaemon) RemoveNetwork(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	_, err := daemon.client.NetworkRemove(ctx, id, client.NetworkRemoveOptions{})
	return err
}

// Rm removes the container with the given id
func (daemon *DockerDaemon) Rm(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	_, err := daemon.client.ContainerRemove(ctx, id, client.ContainerRemoveOptions{
		Force: true,
	})
	if err == nil {
		daemon.store().Remove(id)
	}
	return err
}

// Rmi removes the image with the given name
func (daemon *DockerDaemon) Rmi(name string, force bool) ([]image.DeleteResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := daemon.client.ImageRemove(ctx, name, client.ImageRemoveOptions{
		Force: force,
	})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
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

// StartContainer starts the container with the given id
func (daemon *DockerDaemon) StartContainer(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	if _, err := daemon.client.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		return err
	}
	return daemon.refreshAndWait()
}

// StopContainer stops the container with the given id
func (daemon *DockerDaemon) StopContainer(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	_, err := daemon.client.ContainerStop(ctx, id, client.ContainerStopOptions{})
	if err != nil {
		return err
	}

	return daemon.refreshAndWait()
}

// Top returns Top information for the given container
func (daemon *DockerDaemon) Top(ctx context.Context, id string) (container.TopResponse, error) {
	res, err := daemon.client.ContainerTop(ctx, id, client.ContainerTopOptions{})
	if err != nil {
		return container.TopResponse{}, err
	}
	return container.TopResponse{
		Titles:    res.Titles,
		Processes: res.Processes,
	}, nil
}

// VolumeInspect returns the details of the given volume.
func (daemon *DockerDaemon) VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error) {
	res, err := daemon.client.VolumeInspect(ctx, volumeID, client.VolumeInspectOptions{})
	if err != nil {
		return volume.Volume{}, err
	}
	return res.Volume, nil
}

// VolumeList returns the list of volumes.
func (daemon *DockerDaemon) VolumeList(ctx context.Context) ([]volume.Volume, error) {
	res, err := daemon.client.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

// VolumePrune removes unused volumes.
func (daemon *DockerDaemon) VolumePrune(ctx context.Context) (int, error) {
	res, err := daemon.client.VolumePrune(ctx, client.VolumePruneOptions{})

	return len(res.Report.VolumesDeleted), err
}

// VolumeRemove removes the given volume.
func (daemon *DockerDaemon) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	_, err := daemon.client.VolumeRemove(ctx, volumeID, client.VolumeRemoveOptions{
		Force: force,
	})
	return err
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
func (daemon *DockerDaemon) Version() (*client.ServerVersionResult, error) {
	if daemon.version == nil {
		ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
		defer cancel()
		v, err := daemon.client.ServerVersion(ctx, client.ServerVersionOptions{})
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
	// This loads Docker Version information
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
		func(ctx context.Context, message events.Message) {
			_ = daemon.refreshAndWait()
		})
	return nil
}

func containers(apiClient client.ContainerAPIClient) ([]*Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultConnectionTimeout)
	defer cancel()
	res, err := apiClient.ContainerList(ctx, client.ContainerListOptions{All: true, Size: true})
	if err != nil {
		return nil, fmt.Errorf("retrieve container list: %w", err)
	}

	var cc []*Container
	for i, c := range res.Items {
		details, err := apiClient.ContainerInspect(ctx, c.ID, client.ContainerInspectOptions{})
		if err != nil {
			return nil, fmt.Errorf("inspect container %s: %w", c.ID, err)
		}
		cc = append(cc, &Container{Summary: res.Items[i], Detail: details.Container})
	}
	return cc, nil
}

func images(apiClient client.ImageAPIClient, opts client.ImageListOptions) (client.ImageListResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	return apiClient.ImageList(ctx, opts)
}

func networks(apiClient client.NetworkAPIClient) ([]network.Inspect, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	res, err := apiClient.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	detailedNetworks := make([]network.Inspect, len(res.Items))
	for i, n := range res.Items {
		inspect, err := apiClient.NetworkInspect(ctx, n.ID, client.NetworkInspectOptions{Verbose: true})
		if err != nil {
			return nil, err
		}

		detailedNetworks[i] = inspect.Network
	}

	return detailedNetworks, nil
}

// ComposeProjects returns Docker Compose projects derived from container labels.
func (daemon *DockerDaemon) ComposeProjects() []ComposeProject {
	return AggregateComposeProjects(daemon.Containers(nil, SortByContainerID))
}

// ComposeProjectsWithServices returns Docker Compose projects with their services.
func (daemon *DockerDaemon) ComposeProjectsWithServices() []ProjectWithServices {
	return AggregateComposeAll(daemon.Containers(nil, SortByContainerID))
}

// ComposeServices returns Docker Compose services for a given project.
func (daemon *DockerDaemon) ComposeServices(project string) []ComposeService {
	return AggregateComposeServices(daemon.Containers(nil, SortByContainerID), project)
}

// IsContainerRunning returns true if the given container is running
func IsContainerRunning(container *Container) bool {
	if container != nil {
		return strings.Contains(container.Status, "Up")
	}
	return false
}
