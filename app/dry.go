package app

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/image"
	"github.com/moncho/dry/appui"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	cache "github.com/patrickmn/go-cache"
)

const (
	//TimeBetweenRefresh defines the time that has to pass between dry refreshes
	TimeBetweenRefresh = 30 * time.Second
)

// state tracks dry state
type state struct {
	changed       bool
	filter        drydocker.ContainerFilter
	filterPattern string
	sync.RWMutex
	previousViewMode     viewMode
	showingAllContainers bool
	viewMode             viewMode
	sortMode             drydocker.SortMode
	sortImagesMode       drydocker.SortImagesMode
	sortNetworksMode     drydocker.SortNetworksMode
	activeWidget         appui.Actionable
	node                 string
}

//Dry represents the application.
type Dry struct {
	ui                 *Widgets
	dockerDaemon       drydocker.ContainerDaemon
	dockerEvents       <-chan events.Message
	dockerEventsDone   chan<- struct{}
	imageHistory       []image.HistoryResponseItem
	images             []types.ImageSummary
	info               types.Info
	inspectedContainer types.ContainerJSON
	inspectedImage     types.ImageInspect
	inspectedNetwork   types.NetworkResource
	lastRefresh        time.Time
	networks           []types.NetworkResource
	orderedCids        []string
	output             chan string
	refreshTimerMutex  sync.Locker
	state              *state
	//cache is a potential replacement for state
	cache *cache.Cache
}

//Changed is true if the application state has changed
func (d *Dry) Changed() bool {
	d.state.RLock()
	defer d.state.RUnlock()
	return d.state.changed
}

//changeViewMode changes the view mode of dry
func (d *Dry) changeViewMode(newViewMode viewMode) {
	d.state.Lock()
	defer d.state.Unlock()
	//If the new view is one of the main screens, it must be
	//considered as the view to go back to.
	if newViewMode == Main || newViewMode == Networks || newViewMode == Images {
		d.state.previousViewMode = newViewMode
	}
	d.state.viewMode = newViewMode
	d.state.changed = true
}

//Close closes dry, releasing any resources held by it
func (d *Dry) Close() {
	close(d.dockerEventsDone)
	close(d.output)
}

//ContainerAt returns the container at the given position
func (d *Dry) ContainerAt(position int) *drydocker.Container {

	if position >= 0 {
		containers := d.containerList()
		if len(containers) > position {
			return containers[position]
		}
		return nil
	}
	return nil
}

//ContainerIDAt returns the id of the container at the given position
func (d *Dry) ContainerIDAt(position int) (string, string) {
	c := d.ContainerAt(position)
	if c != nil {
		return c.ID, drydocker.TruncateID(c.ID)
	}
	return "", ""
}
func (d *Dry) containerList() []*drydocker.Container {
	var filter drydocker.ContainerFilter
	if d.state.showingAllContainers {
		filter = drydocker.ContainerFilters.Unfiltered()
	} else {
		filter = drydocker.ContainerFilters.Running()
	}
	containers := d.dockerDaemon.Containers(filter, d.state.sortMode)
	return drydocker.Filter(containers, d.state.filter)
}

//HistoryAt prepares dry to show image history of image at the given positions
func (d *Dry) HistoryAt(position int) {
	if apiImage, err := d.dockerDaemon.ImageAt(position); err == nil {
		d.History(apiImage.ID)
	} else {
		d.appmessage(fmt.Sprintf("<red>Error getting history of image </><white>: %s</>", err.Error()))
	}
}

//History  prepares dry to show image history
func (d *Dry) History(id string) {
	history, err := d.dockerDaemon.History(id)
	if err == nil {
		d.changeViewMode(ImageHistoryMode)
		d.imageHistory = history
	} else {
		d.appmessage(fmt.Sprintf("<red>Error getting history of image </><white>%s: %s</>", id, err.Error()))
	}
}

//InspectAt prepares dry to inspect container at the given position
func (d *Dry) InspectAt(position int) {
	id, _ := d.ContainerIDAt(position)
	if id != "" {
		d.Inspect(id)
	}
}

//Inspect prepares dry to inspect container with the given id
func (d *Dry) Inspect(id string) {
	c, err := d.dockerDaemon.Inspect(id)
	if err == nil {
		d.changeViewMode(InspectMode)
		d.inspectedContainer = c
	} else {
		d.errorMessage(id, "inspecting", err)
	}
}

//InspectImageAt prepares dry to show image information for the image at the given position
func (d *Dry) InspectImageAt(position int) {
	if apiImage, err := d.dockerDaemon.ImageAt(position); err == nil {
		d.InspectImage(apiImage.ID)
	} else {
		d.errorMessage(apiImage.ID, "inspecting image", err)
	}
}

//InspectImage prepares dry to show image information for the image with the given id
func (d *Dry) InspectImage(id string) {
	image, err := d.dockerDaemon.InspectImage(id)
	if err == nil {
		d.changeViewMode(InspectImageMode)
		d.inspectedImage = image
	} else {
		d.errorMessage(id, "inspecting image", err)
	}
}

//InspectNetworkAt prepares dry to show network information for the network at the given position
func (d *Dry) InspectNetworkAt(position int) {
	if network, err := d.dockerDaemon.NetworkAt(position); err == nil {
		d.InspectNetwork(network.ID)
	} else {
		d.errorMessage(network.ID, "inspecting network", err)
	}
}

//InspectNetwork prepares dry to show network information for the network with the given id
func (d *Dry) InspectNetwork(id string) {
	network, err := d.dockerDaemon.NetworkInspect(id)
	if err == nil {
		d.changeViewMode(InspectNetworkMode)
		d.inspectedNetwork = network
	} else {
		d.errorMessage(network.ID, "inspecting network", err)
	}
}

//KillAt the docker container at the given position
func (d *Dry) KillAt(position int) {
	id, _ := d.ContainerIDAt(position)
	if id != "" {
		d.Kill(id)
	}
}

//Kill the docker container with the given id
func (d *Dry) Kill(id string) {

	d.actionMessage(id, "Killing")
	err := d.dockerDaemon.Kill(id)
	if err == nil {
		d.actionMessage(id, "killed")
	} else {
		d.errorMessage(id, "killing", err)
	}

}

//LogsAt retrieves the log of the docker container at the given position
func (d *Dry) LogsAt(position int) (io.ReadCloser, error) {
	id, _ := d.ContainerIDAt(position)
	if id != "" {
		return d.Logs(id)
	}
	return nil, fmt.Errorf("No container found at position %d", position)
}

//Logs retrieves the log of the docker container with the given id
func (d *Dry) Logs(id string) (io.ReadCloser, error) {
	return d.dockerDaemon.Logs(id), nil
}

//NetworkAt returns the network found at the given position.
func (d *Dry) NetworkAt(pos int) (*types.NetworkResource, error) {
	return d.dockerDaemon.NetworkAt(pos)
}

//OuputChannel returns the channel where dry messages are written
func (d *Dry) OuputChannel() <-chan string {
	return d.output
}

//Ok returns the state of dry
func (d *Dry) Ok() (bool, error) {
	return d.dockerDaemon.Ok()
}

//Prune runs docker prune
func (d *Dry) Prune() {
	pr, err := d.dockerDaemon.Prune()
	if err == nil {
		d.cache.Add(pruneReport, pr, 30*time.Second)
	} else {
		d.appmessage(
			fmt.Sprintf(
				"<red>Error running prune. %s</>", err))
	}
}

//PruneReport returns docker prune report, if any available
func (d *Dry) PruneReport() *drydocker.PruneReport {
	if pr, ok := d.cache.Get(pruneReport); ok {
		return pr.(*drydocker.PruneReport)
	}
	return nil
}

//Refresh forces a dry refresh
func (d *Dry) Refresh() {
	d.refreshTimerMutex.Lock()
	defer d.refreshTimerMutex.Unlock()
	d.doRefresh()
	d.resetTimer()
}

func (d *Dry) doRefresh() {
	d.state.Lock()
	defer d.state.Unlock()
	var err error
	switch d.state.viewMode {
	case Main:
		f := func(err error) {
			if err == nil {
				d.state.changed = true
			} else {
				d.appmessage("There was an error refreshing: " + err.Error())
			}
		}
		d.dockerDaemon.Refresh(f)
	case Images:
		err = d.dockerDaemon.RefreshImages()
		d.dockerDaemon.SortImages(d.state.sortImagesMode)

	case Networks:
		err = d.dockerDaemon.RefreshNetworks()
		d.dockerDaemon.SortNetworks(d.state.sortNetworksMode)

	}
	if err != nil {
		d.appmessage("There was an error refreshing: " + err.Error())
	}
}

//RemoveAllStoppedContainers removes all stopped containers
func (d *Dry) RemoveAllStoppedContainers() {
	d.appmessage(fmt.Sprintf("<red>Removing all stopped containers</>"))
	if count, err := d.dockerDaemon.RemoveAllStoppedContainers(); err == nil {
		d.appmessage(fmt.Sprintf("<red>Removed %d stopped containers</>", count))
	} else {
		d.appmessage(
			fmt.Sprintf(
				"<red>Error removing all stopped containers. %s</>", err))
	}
}

//RemoveDanglingImages removes dangling images
func (d *Dry) RemoveDanglingImages() {

	d.appmessage("<red>Removing dangling images</>")
	if count, err := d.dockerDaemon.RemoveDanglingImages(); err == nil {
		d.appmessage(fmt.Sprintf("<red>Removed %d dangling images</>", count))
	} else {
		d.appmessage(
			fmt.Sprintf(
				"<red>Error removing dangling images. %s</>", err))
	}
}

//RemoveImageAt removes the Docker image at the given position
func (d *Dry) RemoveImageAt(position int, force bool) {
	if image, err := d.dockerDaemon.ImageAt(position); err == nil {
		d.RemoveImage(drydocker.ImageID(image.ID), force)
	} else {
		d.appmessage(fmt.Sprintf("<red>Error removing image</>: %s", err.Error()))
	}
}

//RemoveImage removes the Docker image with the given id
func (d *Dry) RemoveImage(id string, force bool) {
	shortID := drydocker.TruncateID(id)
	d.appmessage(fmt.Sprintf("<red>Removing image:</> <white>%s</>", shortID))
	if _, err := d.dockerDaemon.Rmi(id, force); err == nil {
		d.doRefresh()
		d.appmessage(fmt.Sprintf("<red>Removed image:</> <white>%s</>", shortID))
	} else {
		d.appmessage(fmt.Sprintf("<red>Error removing image </><white>%s: %s</>", shortID, err.Error()))
	}
}

//RemoveNetwork removes the Docker network with the given id
func (d *Dry) RemoveNetwork(id string) {
	shortID := drydocker.TruncateID(id)
	d.appmessage(fmt.Sprintf("<red>Removing network:</> <white>%s</>", shortID))
	if err := d.dockerDaemon.RemoveNetwork(id); err == nil {
		d.doRefresh()
		d.appmessage(fmt.Sprintf("<red>Removed network:</> <white>%s</>", shortID))
	} else {
		d.appmessage(fmt.Sprintf("<red>Error network image </><white>%s: %s</>", shortID, err.Error()))
	}
}

func (d *Dry) resetTimer() {
	d.lastRefresh = time.Now()
}

//RestartContainerAt (re)starts the container at the given position
func (d *Dry) RestartContainerAt(position int) {
	id, _ := d.ContainerIDAt(position)
	if id != "" {
		d.RestartContainer(id)
	}
}

//RestartContainer (re)starts the container with the given id
func (d *Dry) RestartContainer(id string) {
	shortID := drydocker.TruncateID(id)
	d.actionMessage(shortID, "Restarting")
	go func() {
		err := d.dockerDaemon.RestartContainer(id)
		if err == nil {
			d.actionMessage(shortID, "Restarted")
		} else {
			d.errorMessage(shortID, "restarting", err)
		}
	}()
}

//RmAt removes the container at the given position
func (d *Dry) RmAt(position int) {
	id, _ := d.ContainerIDAt(position)
	if id != "" {
		d.Rm(id)
	}
}

//Rm removes the container with the given id
func (d *Dry) Rm(id string) {
	shortID := drydocker.TruncateID(id)
	d.actionMessage(shortID, "Removing")
	if err := d.dockerDaemon.Rm(id); err == nil {
		d.actionMessage(shortID, "Removed")
	} else {
		d.errorMessage(shortID, "removing", err)
	}
}

//SetContainerFilter sets a filter for the container list
func (d *Dry) SetContainerFilter(filter string) {
	d.state.Lock()
	defer d.state.Unlock()
	d.state.filterPattern = filter
	//If the given filter pattern is empty the filter is set to null
	//so ContainerIDAt can take the easiest code path.
	if filter != "" {
		d.state.filter = drydocker.ContainerFilters.ByName(filter)
	} else {
		d.state.filter = nil
	}
}

//ShowMainView changes the state of dry to show the main view, main views are
//the container list, the image list or the network list
func (d *Dry) ShowMainView() {
	d.changeViewMode(d.state.previousViewMode)
}

//ShowContainers changes the state of dry to show the container list
func (d *Dry) ShowContainers() {
	d.changeViewMode(Main)
}

//ShowDiskUsage changes the state of dry to show docker disk usage
func (d *Dry) ShowDiskUsage() {
	d.changeViewMode(DiskUsage)
}

//ShowDockerEvents changes the state of dry to show the log of docker events
func (d *Dry) ShowDockerEvents() {
	d.changeViewMode(EventsMode)
}

//ShowHelp changes the state of dry to show the extended help
func (d *Dry) ShowHelp() {
	d.changeViewMode(HelpMode)
}

//ShowImages changes the state of dry to show the list of Docker images reported
//by the daemon
func (d *Dry) ShowImages() {
	if images, err := d.dockerDaemon.Images(); err == nil {
		d.changeViewMode(Images)
		d.images = images
	} else {
		d.appmessage(
			fmt.Sprintf(
				"Could not retrieve image list: %s ", err.Error()))
	}
}

//ShowInfo retrieves Docker Host info.
func (d *Dry) ShowInfo() error {
	info, err := d.dockerDaemon.Info()
	if err == nil {
		d.changeViewMode(InfoMode)
		d.info = info
		return nil
	}
	return err

}

//ShowMonitor changes the state of dry to show the containers monitor
func (d *Dry) ShowMonitor() {
	d.changeViewMode(Monitor)
}

//ShowNetworks changes the state of dry to show the list of Docker networks reported
//by the daemon
func (d *Dry) ShowNetworks() {
	if networks, err := d.dockerDaemon.Networks(); err == nil {
		d.changeViewMode(Networks)
		d.networks = networks
	} else {
		d.appmessage(
			fmt.Sprintf(
				"Could not retrieve network list: %s ", err.Error()))
	}
}

//ShowNodes changes the state of dry to show the node list
func (d *Dry) ShowNodes() {
	d.changeViewMode(Nodes)
}

//ShowServices changes the state of dry to show the service list
func (d *Dry) ShowServices() {
	d.changeViewMode(Services)
}

//ShowTasks changes the state of dry to show the given node's task list
func (d *Dry) ShowTasks(nodeID string) {
	d.state.node = nodeID
	d.changeViewMode(Tasks)

}

//Sort rotates to the next sort mode.
//SortByContainerID -> SortByImage -> SortByStatus -> SortByName -> SortByContainerID
func (d *Dry) Sort() {
	d.state.RLock()
	defer d.state.RUnlock()
	switch d.state.sortMode {
	case drydocker.SortByContainerID:
		d.state.sortMode = drydocker.SortByImage
	case drydocker.SortByImage:
		d.state.sortMode = drydocker.SortByStatus
	case drydocker.SortByStatus:
		d.state.sortMode = drydocker.SortByName
	case drydocker.SortByName:
		d.state.sortMode = drydocker.SortByContainerID
	default:
	}
	d.state.changed = true
}

//SortImages rotates to the next sort mode.
//SortImagesByRepo -> SortImagesByID -> SortImagesByCreationDate -> SortImagesBySize -> SortImagesByRepo
func (d *Dry) SortImages() {
	d.state.RLock()
	defer d.state.RUnlock()
	switch d.state.sortImagesMode {
	case drydocker.SortImagesByRepo:
		d.state.sortImagesMode = drydocker.SortImagesByID
	case drydocker.SortImagesByID:
		d.state.sortImagesMode = drydocker.SortImagesByCreationDate
	case drydocker.SortImagesByCreationDate:
		d.state.sortImagesMode = drydocker.SortImagesBySize
	case drydocker.SortImagesBySize:
		d.state.sortImagesMode = drydocker.SortImagesByRepo

	default:
	}
	d.dockerDaemon.SortImages(d.state.sortImagesMode)
	d.state.changed = true

}

//SortNetworks rotates to the next sort mode.
//SortNetworksByID -> SortNetworksByName -> SortNetworksByDriver
func (d *Dry) SortNetworks() {
	d.state.RLock()
	defer d.state.RUnlock()
	switch d.state.sortNetworksMode {
	case drydocker.SortNetworksByID:
		d.state.sortNetworksMode = drydocker.SortNetworksByName
	case drydocker.SortNetworksByName:
		d.state.sortNetworksMode = drydocker.SortNetworksByDriver
	case drydocker.SortNetworksByDriver:
		d.state.sortNetworksMode = drydocker.SortNetworksByID
	default:
	}
	d.dockerDaemon.SortNetworks(d.state.sortNetworksMode)
	d.state.changed = true
}

func (d *Dry) startDry() {
	go func() {
		for event := range d.dockerEvents {
			//exec_ messages are sent continuously if docker is checking
			//a container health, this events are not shown to the user
			if !strings.Contains(event.Action, "exec_") {
				d.doRefresh()
				d.appmessage(fmt.Sprintf("Docker daemon: %s %s", event.Action, event.ID))
			}
		}
	}()

	go func() {
		for range time.Tick(TimeBetweenRefresh) {
			d.tryRefresh()
		}
	}()
}

//StatsAt get stats of container in the given position until a
//message is sent to the done channel
func (d *Dry) StatsAt(position int) (*drydocker.StatsChannel, error) {
	id, _ := d.ContainerIDAt(position)
	if id != "" {
		return d.Stats(id)
	}
	return nil, fmt.Errorf("Container not found at position %d", position)
}

//Stats get stats of container with the given id until a
//message is sent to the done channel
func (d *Dry) Stats(id string) (*drydocker.StatsChannel, error) {

	c := d.dockerDaemon.ContainerByID(id)

	if c != nil && drydocker.IsContainerRunning(c) {
		return d.dockerDaemon.OpenChannel(c), nil
	}
	d.appmessage(
		fmt.Sprintf("<red>Cannot run stats on stopped container. Id: </><white>%s</>", id))

	return nil, errors.New("Cannot run stats on stopped container.")
}

//StopContainerAt stops the container at the given position
func (d *Dry) StopContainerAt(position int) {
	id, _ := d.ContainerIDAt(position)
	if id != "" {
		d.StopContainer(id)
	}
}

//StopContainer stops the container with the given id
func (d *Dry) StopContainer(id string) {
	shortID := drydocker.TruncateID(id)
	d.actionMessage(shortID, "Stopping")
	go func() {
		if err := d.dockerDaemon.StopContainer(id); err == nil {
			d.actionMessage(shortID, "Stopped")
		} else {
			d.errorMessage(shortID, "stopping", err)
		}
	}()
}

//ToggleShowAllContainers changes between showing running containers and
//showing running and stopped containers.
func (d *Dry) ToggleShowAllContainers() {
	d.state.showingAllContainers = !d.state.showingAllContainers
	if d.state.showingAllContainers {
		d.appmessage("<white>Showing all containers</>")
	} else {
		d.appmessage("<white>Showing running containers</>")
	}
}

//tryRefresh refreshes dry if dry has not been refreshed in the last
//TimeBetweenRefresho
func (d *Dry) tryRefresh() {
	d.refreshTimerMutex.Lock()
	defer d.refreshTimerMutex.Unlock()
	if time.Since(d.lastRefresh) > TimeBetweenRefresh {
		d.resetTimer()
		d.doRefresh()
	}
}

func (d *Dry) appmessage(message string) {
	go func() {
		select {
		case d.output <- message:
		default:
		}
	}()
}

func (d *Dry) actionMessage(cid interface{}, action string) {
	d.appmessage(fmt.Sprintf("<red>%s container with id </><white>%v</>",
		action, cid))
}

func (d *Dry) errorMessage(cid interface{}, action string, err error) {
	d.appmessage(
		fmt.Sprintf(
			"<red>Error %s container </><white>%v. %s</>",
			action, cid, err.Error()))
}

func (d *Dry) viewMode() viewMode {
	d.state.RLock()
	defer d.state.RUnlock()
	return d.state.viewMode
}

func (d *Dry) setChanged(changed bool) {
	d.state.Lock()
	defer d.state.Unlock()
	d.state.changed = changed
}
func newDry(screen *ui.Screen, d *drydocker.DockerDaemon) (*Dry, error) {
	dockerEvents, dockerEventsDone, err := d.Events()
	c := cache.New(5*time.Minute, 30*time.Second)
	if err == nil {

		state := &state{
			changed:              true,
			showingAllContainers: false,
			sortMode:             drydocker.SortByContainerID,
			sortImagesMode:       drydocker.SortImagesByRepo,
			sortNetworksMode:     drydocker.SortNetworksByID,
			viewMode:             Main,
			previousViewMode:     Main,
		}
		d.SortImages(state.sortImagesMode)
		d.SortNetworks(state.sortNetworksMode)
		app := &Dry{}
		app.ui = NewAppUI(d)
		app.state = state
		app.dockerDaemon = d
		app.output = make(chan string)
		app.dockerEvents = dockerEvents
		app.dockerEventsDone = dockerEventsDone
		app.refreshTimerMutex = &sync.Mutex{}
		//first refresh should not happen inmediately after dry creation
		app.lastRefresh = time.Now().Add(TimeBetweenRefresh)
		app.cache = c
		app.startDry()
		return app, nil
	}
	return nil, err
}

//NewDry creates a new dry application
func NewDry(screen *ui.Screen, env *drydocker.Env) (*Dry, error) {
	d, err := drydocker.ConnectToDaemon(env)
	if err != nil {
		return nil, err
	}
	return newDry(screen, d)
}
