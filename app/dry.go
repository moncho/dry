package app

import (
	"bytes"
	"fmt"
	"io"
	"text/template"
	"time"

	godocker "github.com/fsouza/go-dockerclient"
	mdocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	tb "github.com/nsf/termbox-go"
)

//Dry is the application representation.
type Dry struct {
	containerToInspect   *godocker.Container
	dockerDaemon         *mdocker.DockerDaemon
	dockerDaemonRenderer *ui.DockerPs
	header               *header
	State                *AppState
	stats                *mdocker.Stats
	keyboardHelpText     string
}

func (d *Dry) Changed() bool {
	return d.State.changed
}

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

func (d *Dry) Logs(position int) (io.Reader, error) {
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
	case mdocker.SortByContainerID:
		d.State.SortMode = mdocker.SortByImage
	case mdocker.SortByImage:
		d.State.SortMode = mdocker.SortByStatus
	case mdocker.SortByStatus:
		d.State.SortMode = mdocker.SortByName
	case mdocker.SortByName:
		d.State.SortMode = mdocker.SortByContainerID
	default:
	}
}

func (d *Dry) StartContainer(position int) {

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

func (d *Dry) Stats(position int) {
	id, err := d.dockerDaemon.ContainerIDAt(position)
	if err == nil {

		statsC, doneChannel := d.dockerDaemon.Stats(id)
		select {
		case s := <-statsC:
			d.stats = s
			d.State.viewMode = StatsMode
			doneChannel <- true
		}
	}
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

//NewDryApp creates a new dry application
func NewDryApp(screen *ui.Screen) *Dry {
	state := &AppState{
		changed:              true,
		message:              "",
		Paused:               false,
		showingAllContainers: false,
		ShowingHelp:          false,
		SortMode:             mdocker.SortByContainerID,
		viewMode:             Main,
	}
	app := &Dry{}
	newHeader(state)
	app.State = state
	app.header = newHeader(state)
	app.dockerDaemon = mdocker.ConnectToDaemon()
	app.dockerDaemonRenderer = ui.NewDockerRenderer(
		app.dockerDaemon,
		screen.Cursor,
		state.SortMode,
		app.header)
	return app
}

//Not sure if this is the right approach, currently not being used
func newKeyMapping(app *Dry) []KeyPressEvent {
	mapping := []KeyPressEvent{
		KeyPressEvent{
			Key: ui.Key{
				KeyCodes: []rune{'?', 'h', 'H'},
				HelpText: "<b>H:</b><white>Help</>",
			},
			Action: func(app Dry) { app.ShowHelp() },
		},
		KeyPressEvent{
			Key: ui.Key{
				KeyCodes: []rune{'q', 'Q'},
				Keys:     []tb.Key{tb.KeyEsc},
				HelpText: "<b>Q:</b><white>Quit</>",
			},
			Action: func(app Dry) { app.ShowHelp() },
		},
	}
	return mapping
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

	_ = "breakpoint"
	buffer := new(bytes.Buffer)
	h.template.Execute(buffer, vars)
	return buffer.String()
}
