package app

import (
	"fmt"
	"sync"

	"github.com/docker/docker/api/types/events"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//Dry represents the application.
type Dry struct {
	dockerDaemon     drydocker.ContainerDaemon
	dockerEvents     <-chan events.Message
	dockerEventsDone chan<- struct{}
	output           chan string

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

	app := &Dry{}
	widgets = newWidgetRegistry(d)
	viewsToHandlers = initHandlers(app, screen)
	app.dockerDaemon = d
	app.output = make(chan string)
	app.dockerEvents = dockerEvents
	app.dockerEventsDone = dockerEventsDone
	app.startDry()
	return app, nil

}

//NewDry creates a new dry application
func NewDry(screen *ui.Screen, env *drydocker.Env) (*Dry, error) {
	d, err := drydocker.ConnectToDaemon(env)
	if err != nil {
		return nil, err
	}
	return newDry(screen, d)
}
