package app

import (
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types/events"
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
	dockerDaemon     drydocker.ContainerDaemon
	dockerEvents     <-chan events.Message
	dockerEventsDone chan<- struct{}
	output           chan string
	state            *state
	cache            *cache.Cache
}

//SetViewMode changes the view mode of dry
func (d *Dry) SetViewMode(newViewMode viewMode) {
	d.state.Lock()
	defer d.state.Unlock()

	d.state.viewMode = newViewMode
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

//PruneReport returns docker prune report, if any available
func (d *Dry) PruneReport() *drydocker.PruneReport {
	if pr, ok := d.cache.Get(pruneReport); ok {
		return pr.(*drydocker.PruneReport)
	}
	return nil
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
		widgets = newWidgetRegistry(d)
		viewsToHandlers = initHandlers(app, screen)
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
