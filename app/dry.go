package app

import (
	"bytes"
	"fmt"
	"io"
	"text/template"
	"time"

	godocker "github.com/fsouza/go-dockerclient"
	"github.com/moncho/dry/appui"
	drydocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
)

//Dry represents the application.
type Dry struct {
	containerToInspect *godocker.Container
	dockerDaemon       *drydocker.DockerDaemon
	renderer           *appui.DockerPs
	header             *header
	State              *AppState
	stats              *drydocker.Stats
	orderedCids        []string
}

//Changed is true if the application state has changed
func (d *Dry) Changed() bool {
	return d.State.changed
}

//Inspect set dry for inspecting container at the given position
func (d *Dry) Inspect(position int) {
	if id, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		c, err := d.dockerDaemon.Inspect(id)
		if err == nil {
			d.State.viewMode = InspectMode
			d.containerToInspect = c
		} else {
			d.errormessage(id, "inspecting", err)
		}
	} else {
		d.errormessage(id, "inspecting", err)
	}

}

func (d *Dry) Kill(position int) {
	if id, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		err := d.dockerDaemon.Kill(id)
		if err == nil {
			d.appmessage(id, "killed")
		} else {
			d.errormessage(id, "killing", err)
		}
		d.Refresh()
	}

}

func (d *Dry) Logs(position int) (io.ReadCloser, error) {
	id, err := d.dockerDaemon.ContainerIDAt(position)
	if err == nil {
		return d.dockerDaemon.Logs(id), nil
	}
	return nil, err
}

func (d *Dry) Ok() (bool, error) {
	return d.dockerDaemon.Ok()
}

func (d *Dry) Refresh() {
	d.State.changed = true
}

func (d *Dry) Rm(position int) {
	if id, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		if removed := d.dockerDaemon.Rm(id); removed {
			d.Refresh()
			d.appmessage(id, "removed")
		}
	}
}

func (d *Dry) ShowDockerHostInfo() {
	d.State.ShowingHelp = false
	d.State.changed = true
	d.State.viewMode = Main
}

func (d *Dry) ShowHelp() {
	d.State.ShowingHelp = true
	d.State.changed = true
	d.State.viewMode = HelpMode
}

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

}

func (d *Dry) StartContainer(position int) {
	_ = "breakpoint"
	if id, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		err := d.dockerDaemon.RestartContainer(id)
		if err == nil {
			d.appmessage(id, "restarted")
		} else {
			d.errormessage(id, "restarting", err)
		}
		d.Refresh()
	}
}

//Stats get stats of container in the given position until a
//message is sent to the done channel
func (d *Dry) Stats(position int) (chan<- bool, error) {
	id, err := d.dockerDaemon.ContainerIDAt(position)
	if err == nil {
		done := make(chan bool, 1)
		statsC, dockerDoneChannel, err := d.dockerDaemon.Stats(id)
		if err == nil {
			go func() {
			loop:
				for {
					select {
					case s := <-statsC:
						d.stats = s
						d.State.viewMode = StatsMode
					case <-done:
						dockerDoneChannel <- true
						close(done)
						break loop
					}
				}
			}()
			return done, nil
		}
	}
	return nil, err
}

func (d *Dry) StopContainer(position int) {
	if id, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		err := d.dockerDaemon.StopContainer(id)
		if err == nil {
			d.appmessage(id, "stopped")
		} else {
			d.errormessage(id, "stopping", err)
		}
		d.Refresh()
	}
}

func (d *Dry) ToggleShowAllContainers() {
	d.State.showingAllContainers = !d.State.showingAllContainers
	d.State.changed = true
}

func (d *Dry) appmessage(cid string, action string) {
	d.State.message = fmt.Sprintf("<red>Container with id </><white>%s</> <red>%s</>",
		cid,
		action)
}
func (d *Dry) cleanStats() {
	d.stats = nil
}

func (d *Dry) errormessage(cid string, action string, err error) {
	d.State.message = err.Error()
}

func newDry(screen *ui.Screen, d *drydocker.DockerDaemon, err error) (*Dry, error) {
	if err == nil {
		state := &AppState{
			changed:              true,
			message:              "",
			Paused:               false,
			showingAllContainers: false,
			ShowingHelp:          false,
			SortMode:             drydocker.SortByContainerID,
			viewMode:             Main,
		}
		d.Sort(state.SortMode)
		app := &Dry{}
		//newHeader(state)
		app.State = state
		app.header = newHeader(state)
		app.dockerDaemon = d
		app.renderer = appui.NewDockerRenderer(
			app.dockerDaemon,
			screen.Cursor,
			state.SortMode,
			app.header)
		return app, nil
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
	d, err := drydocker.ConnectToGivenDaemon(env)
	return newDry(screen, d, err)
}

//header
type header struct {
	template *template.Template
	appState *AppState
}

func newHeader(state *AppState) *header {
	return &header{
		buildHeaderTemplate(),
		state,
	}
}
func buildHeaderTemplate() *template.Template {
	markup := `{{.AppMessage}}<right><white>{{.Now}}</></right>`
	return template.Must(template.New(`header`).Parse(markup))
}

func (h *header) Render() string {
	vars := struct {
		Now        string // Current timestamp.
		AppMessage string
	}{
		time.Now().Format(`3:04:05pm PST`),
		h.appState.Render(),
	}

	buffer := new(bytes.Buffer)
	h.template.Execute(buffer, vars)
	return buffer.String()
}
