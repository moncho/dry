package app

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/pkg/stringid"
	godocker "github.com/fsouza/go-dockerclient"
	"github.com/moncho/dry/appui"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

const (
	//TimeBetweenRefresh defines the time that has to pass between dry refreshes
	TimeBetweenRefresh = 10 * time.Second
)

// state tracks dry state
type state struct {
	changed              bool
	showingAllContainers bool
	viewMode             viewMode
	previousViewMode     viewMode
	SortMode             drydocker.SortMode
	SortImagesMode       drydocker.SortImagesMode
	SortNetworksMode     drydocker.SortNetworksMode
	viewMutex            sync.Locker
}

//Dry represents the application.
type Dry struct {
	dockerDaemon       drydocker.ContainerDaemon
	dockerEvents       chan *godocker.APIEvents
	imageHistory       []godocker.ImageHistory
	images             []godocker.APIImages
	info               *godocker.Env
	inspectedContainer *godocker.Container
	inspectedImage     *godocker.Image
	inspectedNetwork   *godocker.Network
	lastRefresh        time.Time
	networks           []godocker.Network
	orderedCids        []string
	output             chan string
	refreshTimeMutex   sync.Locker
	renderer           *appui.DockerPs
	state              *state
	stats              *drydocker.Stats
}

//Changed is true if the application state has changed
func (d *Dry) Changed() bool {
	return d.state.changed
}

//changeViewMode changes the view mode of dry
func (d *Dry) changeViewMode(newViewMode viewMode) {
	d.state.viewMutex.Lock()
	defer d.state.viewMutex.Unlock()
	if newViewMode == Main || newViewMode == Networks || newViewMode == Images {
		d.state.previousViewMode = newViewMode
	} else {
		d.state.previousViewMode = d.state.viewMode
	}
	d.state.previousViewMode = d.state.viewMode
	d.state.viewMode = newViewMode
	d.state.changed = true
}

//Close closes dry, releasing any resources held by it
func (d *Dry) Close() {
	d.dockerDaemon.StopEventChannel(d.dockerEvents)
	close(d.output)
}

//History  prepares dry to show image history
func (d *Dry) History(position int) {
	if apiImage, err := d.dockerDaemon.ImageAt(position); err == nil {
		history, err := d.dockerDaemon.History(apiImage.ID)
		if err == nil {
			d.changeViewMode(ImageHistoryMode)
			d.imageHistory = history
		} else {
			d.appmessage(fmt.Sprintf("<red>Error getting history of image </><white>%s: %s</>", apiImage.ID, err.Error()))
		}
	} else {
		d.appmessage(fmt.Sprintf("<red>Error getting history of image </><white>: %s</>", err.Error()))
	}

}

//Inspect prepares dry to inspect container at the given position
func (d *Dry) Inspect(position int) {
	if id, shortID, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		c, err := d.dockerDaemon.Inspect(id)
		if err == nil {
			d.changeViewMode(InspectMode)
			d.inspectedContainer = c
		} else {
			d.errormessage(shortID, "inspecting", err)
		}
	} else {
		d.errormessage(shortID, "inspecting", err)
	}
}

//InspectImage prepares dry to show image information for the image at the given position
func (d *Dry) InspectImage(position int) {

	if apiImage, err := d.dockerDaemon.ImageAt(position); err == nil {
		image, err := d.dockerDaemon.InspectImage(apiImage.ID)
		if err == nil {
			d.changeViewMode(InspectImageMode)
			d.inspectedImage = image
		} else {
			d.errormessage(apiImage.ID, "inspecting image", err)
		}
	} else {
		d.errormessage(apiImage.ID, "inspecting image", err)
	}
}

//InspectNetwork prepares dry to show network information for the network at the given position
func (d *Dry) InspectNetwork(position int) {

	if network, err := d.dockerDaemon.NetworkAt(position); err == nil {
		network, err := d.dockerDaemon.NetworkInspect(network.ID)
		if err == nil {
			d.changeViewMode(InspectNetworkMode)
			d.inspectedNetwork = network
		} else {
			d.errormessage(network.ID, "inspecting network", err)
		}
	} else {
		d.errormessage(network.ID, "inspecting network", err)
	}
}

//Kill the docker container at the given position
func (d *Dry) Kill(position int) {
	if id, shortID, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		d.actionmessage(shortID, "Killing")
		err := d.dockerDaemon.Kill(id)
		if err == nil {
			d.actionmessage(shortID, "killed")
		} else {
			d.errormessage(shortID, "killing", err)
		}
	}
}

//Logs retrieves the log of the docker container at the given position
func (d *Dry) Logs(position int) (io.ReadCloser, error) {
	id, _, err := d.dockerDaemon.ContainerIDAt(position)
	if err == nil {
		return d.dockerDaemon.Logs(id), nil
	}
	return nil, err
}

//OuputChannel returns the channel where dry messages are written
func (d *Dry) OuputChannel() <-chan string {
	return d.output
}

//Ok returns the state of dry
func (d *Dry) Ok() (bool, error) {
	return d.dockerDaemon.Ok()
}

//Refresh forces a dry refresh
func (d *Dry) Refresh() {
	d.refreshTimeMutex.Lock()
	defer d.refreshTimeMutex.Unlock()
	d.doRefresh()
	d.resetTimer()
}

func (d *Dry) doRefresh() {
	d.state.changed = true
	err := d.dockerDaemon.Refresh(d.state.showingAllContainers)
	if err != nil {
		d.appmessage("There was an error refreshing: " + err.Error())
	}
	err = d.dockerDaemon.RefreshImages()
	if err != nil {
		d.appmessage("There was an error refreshing: " + err.Error())
	}

	err = d.dockerDaemon.RefreshNetworks()
	if err != nil {
		d.appmessage("There was an error refreshing: " + err.Error())
	}

}

//RemoveAllStoppedContainers removes all stopped containers
func (d *Dry) RemoveAllStoppedContainers() {
	d.appmessage(fmt.Sprintf("<red>Removing all stopped containers</>"))
	if err := d.dockerDaemon.RemoveAllStoppedContainers(); err == nil {
		d.appmessage(fmt.Sprintf("<red>Removed all stopped containers</>"))
	} else {
		d.appmessage(
			fmt.Sprintf(
				"<red>Error removing all stopped containers. %s</>", err))
	}
}

//RemoveImage removes the Docker image at the given position
func (d *Dry) RemoveImage(position int) {
	if image, err := d.dockerDaemon.ImageAt(position); err == nil {
		id := drydocker.ImageID(image.ID)
		shortID := stringid.TruncateID(id)
		d.appmessage(fmt.Sprintf("<red>Removing image:</> <white>%s</>", shortID))
		if err := d.dockerDaemon.Rmi(id); err == nil {
			d.appmessage(fmt.Sprintf("<red>Removed image:</> <white>%s</>", shortID))
		} else {
			d.appmessage(fmt.Sprintf("<red>Error removing image </><white>%s: %s</>", shortID, err.Error()))
		}
	} else {
		d.appmessage(fmt.Sprintf("<red>Error removing image</>: %s", err.Error()))
	}
}

func (d *Dry) resetTimer() {
	d.lastRefresh = time.Now()
}

//Rm removes the container at the given position
func (d *Dry) Rm(position int) {
	if id, shortID, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		d.actionmessage(shortID, "Removing")
		if err := d.dockerDaemon.Rm(id); err == nil {
			d.actionmessage(shortID, "Removed")
		} else {
			d.errormessage(shortID, "removing", err)
		}
	} else {
		d.errormessage(id, "removing", err)
	}
}

//ShowMainView changes the state of dry to show the main view, main views are
//either the container list or the image list
func (d *Dry) ShowMainView() {
	d.changeViewMode(d.state.previousViewMode)
}

//ShowContainers changes the state of dry to show the container list
func (d *Dry) ShowContainers() {
	d.changeViewMode(Main)
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

//Sort rotates to the next sort mode.
//SortByContainerID -> SortByImage -> SortByStatus -> SortByName -> SortByContainerID
func (d *Dry) Sort() {
	switch d.state.SortMode {
	case drydocker.SortByContainerID:
		d.state.SortMode = drydocker.SortByImage
	case drydocker.SortByImage:
		d.state.SortMode = drydocker.SortByStatus
	case drydocker.SortByStatus:
		d.state.SortMode = drydocker.SortByName
	case drydocker.SortByName:
		d.state.SortMode = drydocker.SortByContainerID
	default:
	}
	d.dockerDaemon.Sort(d.state.SortMode)
	d.state.changed = true
}

//SortImages rotates to the next sort mode.
//SortImagesByRepo -> SortImagesByID -> SortImagesByCreationDate -> SortImagesBySize -> SortImagesByRepo
func (d *Dry) SortImages() {
	switch d.state.SortImagesMode {
	case drydocker.SortImagesByRepo:
		d.state.SortImagesMode = drydocker.SortImagesByID
	case drydocker.SortImagesByID:
		d.state.SortImagesMode = drydocker.SortImagesByCreationDate
	case drydocker.SortImagesByCreationDate:
		d.state.SortImagesMode = drydocker.SortImagesBySize
	case drydocker.SortImagesBySize:
		d.state.SortImagesMode = drydocker.SortImagesByRepo

	default:
	}
	d.dockerDaemon.SortImages(d.state.SortImagesMode)
	d.state.changed = true

}

//SortNetworks rotates to the next sort mode.
//SortNetworksByID -> SortNetworksByName -> SortNetworksByDriver
func (d *Dry) SortNetworks() {
	switch d.state.SortNetworksMode {
	case drydocker.SortNetworksByID:
		d.state.SortNetworksMode = drydocker.SortNetworksByName
	case drydocker.SortNetworksByName:
		d.state.SortNetworksMode = drydocker.SortNetworksByDriver
	case drydocker.SortNetworksByDriver:
		d.state.SortNetworksMode = drydocker.SortNetworksByID
	default:
	}
	d.dockerDaemon.SortNetworks(d.state.SortNetworksMode)
	d.state.changed = true
}

func (d *Dry) startDry() {
	go func() {
		//The event is not relevant, dry must refresh
		for range d.dockerEvents {
			d.Refresh()
		}
	}()

	go func() {
		for range time.Tick(15 * time.Second) {
			d.tryRefresh()
		}
	}()
}

//StartContainer (re)starts the container at the given position
func (d *Dry) StartContainer(position int) {
	if id, shortID, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		d.actionmessage(shortID, "Restarting")
		go func() {
			err := d.dockerDaemon.RestartContainer(id)
			if err == nil {
				d.actionmessage(shortID, "Restarted")
			} else {
				d.errormessage(shortID, "restarting", err)
			}
		}()
	} else {
		d.errormessage(shortID, "restarting", err)
	}

}

//Stats get stats of container in the given position until a
//message is sent to the done channel
func (d *Dry) Stats(position int) (chan<- bool, <-chan error, error) {
	id, _, err := d.dockerDaemon.ContainerIDAt(position)
	if err == nil {
		if d.dockerDaemon.IsContainerRunning(id) {
			done := make(chan bool, 1)
			statsC, dockerDoneChannel, errC := d.dockerDaemon.Stats(id)
			if err == nil {
				go func() {
					for {
						select {
						case s := <-statsC:
							d.stats = s
						case <-done:
							dockerDoneChannel <- true
							return
						}
					}
				}()
				d.changeViewMode(StatsMode)
				return done, errC, nil
			}
			dockerDoneChannel <- true
		}
		d.appmessage(
			fmt.Sprintf("<red>Cannot run stats on stopped container. Id: </><white>%s</>", id))
		err = errors.New("Cannot run stats on stopped container.")
	}
	return nil, nil, err
}

//StopContainer stops the container at the given position
func (d *Dry) StopContainer(position int) {
	if id, shortID, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		d.actionmessage(shortID, "Stopping")
		go func() {
			if err := d.dockerDaemon.StopContainer(id); err == nil {
				d.actionmessage(shortID, "Stopped")
			} else {
				d.errormessage(shortID, "stopping", err)
			}
		}()
	}
}

//ToggleShowAllContainers changes between showing running containers and
//showing running and stopped containers.
func (d *Dry) ToggleShowAllContainers() {
	d.state.showingAllContainers = !d.state.showingAllContainers
	d.Refresh()
	if d.state.showingAllContainers {
		d.appmessage("<white>Showing all containers</>")
	} else {
		d.appmessage("<white>Showing running containers</>")
	}
}

//tryRefresh refreshes dry if dry has not been refreshed in the last
//TimeBetweenRefresh
func (d *Dry) tryRefresh() {
	d.refreshTimeMutex.Lock()
	defer d.refreshTimeMutex.Unlock()
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

func (d *Dry) actionmessage(cid string, action string) {
	d.appmessage(fmt.Sprintf("<red>%s container with id </><white>%s</>",
		action, cid))
}
func (d *Dry) cleanStats() {
	d.stats = nil
}

func (d *Dry) errormessage(cid string, action string, err error) {
	d.appmessage(
		fmt.Sprintf(
			"<red>Error %s container </><white>%s. %s</>",
			action, cid, err.Error()))
}

func (d *Dry) viewMode() viewMode {
	d.state.viewMutex.Lock()
	defer d.state.viewMutex.Unlock()
	return d.state.viewMode
}
func newDry(screen *ui.Screen, d *drydocker.DockerDaemon, err error) (*Dry, error) {
	if err == nil {
		dockerEvents, err := d.Events()
		if err == nil {

			state := &state{
				changed:              true,
				showingAllContainers: false,
				SortMode:             drydocker.SortByContainerID,
				SortImagesMode:       drydocker.SortImagesByRepo,
				SortNetworksMode:     drydocker.SortNetworksByID,
				viewMode:             Main,
				previousViewMode:     Main,
				viewMutex:            &sync.Mutex{},
			}
			d.Sort(state.SortMode)
			app := &Dry{}
			app.state = state
			app.dockerDaemon = d
			app.renderer = appui.NewDockerPsRenderer(
				app.dockerDaemon,
				screen.Cursor,
				state.SortMode)
			app.output = make(chan string)
			app.dockerEvents = dockerEvents
			app.refreshTimeMutex = &sync.Mutex{}
			app.resetTimer()
			app.startDry()
			return app, nil
		}
		return nil, err
	}
	return nil, err
}

//NewDryApp creates a new dry application
func NewDryApp(screen *ui.Screen) (*Dry, error) {
	d, err := drydocker.ConnectToDaemon()
	return newDry(screen, d, err)
}

//NewDryAppWithDockerEnv creates a new dry application
func NewDryAppWithDockerEnv(screen *ui.Screen, env *drydocker.DockerEnv) (*Dry, error) {
	d, err := drydocker.ConnectToDaemonUsingEnv(env)
	return newDry(screen, d, err)
}
