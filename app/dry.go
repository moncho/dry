package app

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/swarm"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	cache "github.com/patrickmn/go-cache"
)

// state tracks dry state
type state struct {
	sync.RWMutex
	previousViewMode viewMode
	viewMode         viewMode
}

//Dry represents the application.
type Dry struct {
	widgetRegistry   *WidgetRegistry
	dockerDaemon     drydocker.ContainerDaemon
	dockerEvents     <-chan events.Message
	dockerEventsDone chan<- struct{}
	info             types.Info
	output           chan string
	state            *state
	//cache is a potential replacement for state
	cache *cache.Cache
}

//changeViewMode changes the view mode of dry and refreshes the screen
func (d *Dry) changeViewMode(newViewMode viewMode) {
	d.SetViewMode(newViewMode)
	refreshScreen()
}

//SetViewMode changes the view mode of dry
func (d *Dry) SetViewMode(newViewMode viewMode) {
	d.state.Lock()
	defer d.state.Unlock()
	//If the new view is one of the main screens, it must be
	//considered as the view to go back to.
	if newViewMode.isMainScreen() {
		d.state.previousViewMode = newViewMode
	}
	d.state.viewMode = newViewMode
}

//Close closes dry, releasing any resources held by it
func (d *Dry) Close() {
	close(d.dockerEventsDone)
	close(d.output)
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

//Logs retrieves the log of the docker container with the given id
func (d *Dry) Logs(id string) (io.ReadCloser, error) {
	return d.dockerDaemon.Logs(id), nil
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

//RemoveImage removes the Docker image with the given id
func (d *Dry) RemoveImage(id string, force bool) {
	shortID := drydocker.TruncateID(id)
	d.appmessage(fmt.Sprintf("<red>Removing image:</> <white>%s</>", shortID))
	if _, err := d.dockerDaemon.Rmi(id, force); err == nil {
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
		d.appmessage(fmt.Sprintf("<red>Removed network:</> <white>%s</>", shortID))
	} else {
		d.appmessage(fmt.Sprintf("<red>Error network image </><white>%s: %s</>", shortID, err.Error()))
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

//ServiceInspect returns information about the service with the given ID
func (d *Dry) ServiceInspect(id string) (*swarm.Service, error) {
	return d.dockerDaemon.Service(id)
}

//ServiceLogs retrieves the log of the service with the given id
func (d *Dry) ServiceLogs(id string) (io.ReadCloser, error) {
	return d.dockerDaemon.ServiceLogs(id)
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
	d.changeViewMode(Images)
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
	d.changeViewMode(Networks)
}

//ShowNodes changes the state of dry to show the node list
func (d *Dry) ShowNodes() {
	d.changeViewMode(Nodes)
}

//ShowServices changes the state of dry to show the service list
func (d *Dry) ShowServices() {
	d.changeViewMode(Services)
}

//ShowTasks changes the state of dry to show the stack list
func (d *Dry) ShowStacks() {
	d.changeViewMode(Stacks)
}

//ShowServiceTasks changes the state of dry to show the given service task list
func (d *Dry) ShowServiceTasks(serviceID string) {
	d.widgetRegistry.ServiceTasks.ForService(serviceID)
	d.changeViewMode(ServiceTasks)
}

//ShowStackTasks changes the state of dry to show the given stack task list
func (d *Dry) ShowStackTasks(task string) {
	d.widgetRegistry.StackTasks.ForStack(task)
	d.changeViewMode(StackTasks)
}

//ShowTasks changes the state of dry to show the given node task list
func (d *Dry) ShowTasks(nodeID string) {
	d.widgetRegistry.NodeTasks.ForNode(nodeID)
	d.changeViewMode(Tasks)
}

func (d *Dry) startDry() {
	de := dockerEventsListener{d}
	de.init()
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

func newDry(screen *ui.Screen, d *drydocker.DockerDaemon) (*Dry, error) {
	dockerEvents, dockerEventsDone, err := d.Events()
	c := cache.New(5*time.Minute, 30*time.Second)
	if err == nil {

		state := &state{
			viewMode:         Main,
			previousViewMode: Main,
		}
		app := &Dry{}
		app.widgetRegistry = NewWidgetRegistry(d)
		app.state = state
		app.dockerDaemon = d
		app.output = make(chan string)
		app.dockerEvents = dockerEvents
		app.dockerEventsDone = dockerEventsDone
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
