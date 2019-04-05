package app

import (
	"fmt"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/events"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//Dry resources and state
type Dry struct {
	dockerDaemon     drydocker.ContainerDaemon
	dockerEvents     <-chan events.Message
	dockerEventsDone chan<- struct{}
	output           chan string
	screen           *ui.Screen

	sync.RWMutex
	view viewMode
}

//Close closes dry, releasing any resources held by it
func (d *Dry) Close() {
	close(d.dockerEventsDone)
	close(d.output)
}

//OuputChannel returns the channel where dry messages are written
func (d *Dry) OuputChannel() <-chan string {
	return d.output
}

//Ok returns the state of dry
func (d *Dry) Ok() (bool, error) {
	return d.dockerDaemon.Ok()
}

//ViewMode changes the view mode of dry
func (d *Dry) ViewMode(v viewMode) {
	d.Lock()
	defer d.Unlock()

	d.view = v
}

func (d *Dry) showDockerEvents() {
	go func() {
		for event := range d.dockerEvents {
			//exec_ messages are sent continuously if docker is checking
			//a container's health, so they are ignored
			if strings.Contains(event.Action, "exec_") {
				continue
			}
			//top messages are sent continuously on monitor mode, ignored
			if strings.Contains(event.Action, "top") {
				continue
			}
			d.message(fmt.Sprintf("Docker: %s %s", event.Action, event.ID))
		}
	}()
}

//message publishes the given message
func (d *Dry) message(message string) {
	select {
	case d.output <- message:
	default:
	}
}

func (d *Dry) actionMessage(cid interface{}, action string) {
	d.message(fmt.Sprintf("<red>%s container with id </><white>%v</>",
		action, cid))
}

func (d *Dry) errorMessage(cid interface{}, action string, err error) {
	d.message(
		fmt.Sprintf(
			"%s", err.Error()))
}

func (d *Dry) viewMode() viewMode {
	d.RLock()
	defer d.RUnlock()
	return d.view
}

func newDry(screen *ui.Screen, d *drydocker.DockerDaemon) (*Dry, error) {
	dockerEvents, dockerEventsDone, err := d.Events()
	if err != nil {
		return nil, err
	}

	dry := &Dry{}
	widgets = initRegistry(d)
	viewsToHandlers = initHandlers(dry, screen)
	dry.dockerDaemon = d
	dry.output = make(chan string)
	dry.dockerEvents = dockerEvents
	dry.dockerEventsDone = dockerEventsDone
	dry.screen = screen
	dry.showDockerEvents()
	return dry, nil

}

//NewDry creates a new dry application
func NewDry(screen *ui.Screen, env *drydocker.Env) (*Dry, error) {
	d, err := drydocker.ConnectToDaemon(env)
	if err != nil {
		return nil, err
	}
	return newDry(screen, d)
}
