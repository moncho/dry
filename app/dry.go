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
	output             chan string
}

//Changed is true if the application state has changed
func (d *Dry) Changed() bool {
	return d.State.changed
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
		d.Refresh()
	}

}

//Logs the docker container at the given position
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
	d.State.changed = true
}

//Rm removes the container at the given position
func (d *Dry) Rm(position int) {
	if id, shortID, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		d.actionmessage(shortID, "Removing")
		if err := d.dockerDaemon.Rm(id); err == nil {
			d.Refresh()
			d.actionmessage(shortID, "Removed")
		} else {
			d.errormessage(shortID, "removing", err)
		}
	} else {
		d.errormessage(id, "removing", err)
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
	if id, shortID, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		d.actionmessage(shortID, "Restarting")
		go func() {
			err := d.dockerDaemon.RestartContainer(id)
			if err == nil {
				//d.Refresh()
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
	return nil, nil, err
}

func (d *Dry) StopContainer(position int) {
	if id, shortID, err := d.dockerDaemon.ContainerIDAt(position); err == nil {
		d.actionmessage(shortID, "Stopping")
		if err := d.dockerDaemon.StopContainer(id); err == nil {
			d.actionmessage(shortID, "Stopped")
			//d.Refresh()
		} else {
			d.errormessage(shortID, "stopping", err)
		}
	}
}

func (d *Dry) ToggleShowAllContainers() {
	d.State.showingAllContainers = !d.State.showingAllContainers
	d.State.changed = true
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
	_ = "breakpoint"
	if err == nil {
		state := &AppState{
			changed:              true,
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
		app.output = make(chan string)
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
	d, err := drydocker.ConnectToDaemonUsingEnv(env)
	return newDry(screen, d, err)
}

// ------------------------
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
