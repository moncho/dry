package app

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	godocker "github.com/fsouza/go-dockerclient"
	"github.com/moncho/dry/appui"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

const (
	//TimeBetweenRefresh defines the time that has to pass between dry refreshes
	TimeBetweenRefresh = 10 * time.Second
)

//Dry represents the application.
type Dry struct {
	containerToInspect *godocker.Container
	dockerDaemon       drydocker.ContainerDaemon
	info               *godocker.Env
	renderer           *appui.DockerPs
	State              *State
	stats              *drydocker.Stats
	orderedCids        []string
	output             chan string
	dockerEvents       chan *godocker.APIEvents
	lastRefresh        time.Time
	refreshTimeMutex   sync.Locker
}

//Changed is true if the application state has changed
func (d *Dry) Changed() bool {
	return d.State.changed
}

//Close closes dry, releasing any resources held by it
func (d *Dry) Close() {
	d.dockerDaemon.StopEventChannel(d.dockerEvents)
	close(d.output)
	close(d.dockerEvents)
}

//Inspect set dry for inspecting container at the given position
func (d *Dry) Inspect(position int) {
	if id, shortID, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		c, err := d.dockerDaemon.Inspect(id)
		if err == nil {
			d.State.viewMode = InspectMode
			d.containerToInspect = c
		} else {
			d.errormessage(shortID, "inspecting", err)
		}
	} else {
		d.errormessage(shortID, "inspecting", err)
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
	d.State.changed = true
	d.dockerDaemon.Refresh(d.State.showingAllContainers)
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

//ShowContainers changes the state of dry to show the container list
func (d *Dry) ShowContainers() {
	d.State.changed = true
	d.State.viewMode = Main
}

//ShowHelp changes the state of dry to show the extended help
func (d *Dry) ShowHelp() {
	d.State.viewMode = HelpMode
}

//ShowInfo retrieves Docker Host info.
func (d *Dry) ShowInfo() error {
	info, err := d.dockerDaemon.Info()
	if err == nil {
		d.State.viewMode = InfoMode
		d.info = info
		return nil
	}
	return err

}

//Sort rotates to the next sort mode.
//SortByContainerID -> SortByImage -> SortByStatus -> SortByName -> SortByContainerID
func (d *Dry) Sort() {
	switch d.State.SortMode {
	case drydocker.SortByContainerID:
		d.State.SortMode = drydocker.SortByImage
	case drydocker.SortByImage:
		d.State.SortMode = drydocker.SortByStatus
	case drydocker.SortByStatus:
		d.State.SortMode = drydocker.SortByName
	case drydocker.SortByName:
		d.State.SortMode = drydocker.SortByContainerID
	default:
	}
	d.dockerDaemon.Sort(d.State.SortMode)
	d.State.changed = true

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
							d.State.viewMode = StatsMode
						case <-done:
							dockerDoneChannel <- true
							return
						}
					}
				}()
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
	d.State.showingAllContainers = !d.State.showingAllContainers
	d.Refresh()
	if d.State.showingAllContainers {
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

func newDry(screen *ui.Screen, d *drydocker.DockerDaemon, err error) (*Dry, error) {
	if err == nil {
		dockerEvents, err := d.Events()
		if err == nil {

			state := &State{
				changed:              true,
				Paused:               false,
				showingAllContainers: false,
				SortMode:             drydocker.SortByContainerID,
				viewMode:             Main,
			}
			d.Sort(state.SortMode)
			app := &Dry{}
			app.State = state
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
