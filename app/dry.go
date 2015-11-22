package app

import (
	"bytes"
	"fmt"
	"io"
	"text/template"
	"time"

	mdocker "github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	tb "github.com/nsf/termbox-go"
)

//Dry is the application representation.
type Dry struct {
	dockerDaemon         *mdocker.DockerDaemon
	dockerDaemonRenderer *ui.DockerPs
	header               *header
	screen               *ui.Screen
	State                *AppState
	stats                *mdocker.Stats
	keyboardHelpText     string
}

func (m *Dry) Changed() bool {
	return m.State.changed
}

func (m *Dry) Kill(position int) {
	if id, err := m.dockerDaemon.ContainerIDAt(position); err == nil {
		err := m.dockerDaemon.Kill(id)
		if err == nil {
			m.appmessage(id, "killed")
		} else {
			m.errormessage(id, "killing", err)
		}
		m.Refresh()
	}

}

func (m *Dry) Logs(position int) (io.Reader, error) {
	id, err := m.dockerDaemon.ContainerIDAt(position)
	if err == nil {
		return m.dockerDaemon.Logs(id), nil
	}
	return nil, err
}

func (m *Dry) Ok() (bool, error) {
	return m.dockerDaemon.Ok()
}

func (m *Dry) Refresh() {
	m.State.changed = true
}

func (m *Dry) Render() {
	if m.State.ShowingHelp {
		m.screen.Clear()
		m.screen.Render(0, help)
		m.State.ShowingHelp = false
	} else if m.State.showingStats && m.stats != nil {
		m.screen.Render(0, ui.NewDockerStatsRenderer(m.stats).Render())
	} else {
		m.dockerDaemon.Refresh(m.State.showingAllContainers)
		m.dockerDaemonRenderer.SortMode(m.State.SortMode)
		m.screen.Render(0, m.dockerDaemonRenderer.Render())
		m.screen.RenderLine(0, 0, `<right><white>`+time.Now().Format(`3:04:05pm PST`)+`</></right>`)
		m.screen.RenderLine(0, m.screen.Height-1, keyMappings)
		m.State.changed = false
	}
	m.screen.Flush()
}

func (m *Dry) Rm(position int) {
	if id, err := m.dockerDaemon.ContainerIDAt(position); err == nil {
		if removed := m.dockerDaemon.Rm(id); removed {
			m.Refresh()
			m.appmessage(id, "removed")
		}
	}
}

func (m *Dry) ShowDockerInfo() {
	m.State.ShowingHelp = false
	m.State.changed = true
}

func (m *Dry) ShowHelp() {
	m.State.ShowingHelp = true
}

func (m *Dry) Sort() {
	switch m.State.SortMode {
	case mdocker.SortByContainerID:
		m.State.SortMode = mdocker.SortByImage
	case mdocker.SortByImage:
		m.State.SortMode = mdocker.SortByStatus
	case mdocker.SortByStatus:
		m.State.SortMode = mdocker.SortByName
	case mdocker.SortByName:
		m.State.SortMode = mdocker.SortByContainerID
	default:
	}
}

func (m *Dry) StartContainer(position int) {

	if id, err := m.dockerDaemon.ContainerIDAt(position); err == nil {
		err := m.dockerDaemon.RestartContainer(id)
		if err == nil {
			m.appmessage(id, "restarted")
		} else {
			m.errormessage(id, "restarting", err)
		}
		m.Refresh()
	}
}

func (m *Dry) Stats(position int) {
	id, err := m.dockerDaemon.ContainerIDAt(position)
	if err == nil {

		statsC, doneChannel := m.dockerDaemon.Stats(id)
		select {
		case s := <-statsC:
			m.stats = s
			m.State.showingStats = true
			doneChannel <- true
		}
	}
}

func (m *Dry) StopContainer(position int) {
	if id, err := m.dockerDaemon.ContainerIDAt(position); err == nil {
		err := m.dockerDaemon.StopContainer(id)
		if err == nil {
			m.appmessage(id, "stopped")
		} else {
			m.errormessage(id, "stopping", err)
		}
		m.Refresh()
	}
}

func (m *Dry) ToggleShowAllContainers() {
	m.State.showingAllContainers = !m.State.showingAllContainers
	m.State.changed = true
}

func (m *Dry) appmessage(cid string, action string) {
	m.State.message = fmt.Sprintf("<red>Container with id </><white>%s</> <red>%s</>",
		cid,
		action)
}
func (m *Dry) cleanStats() {
	m.stats = nil
}

func (m *Dry) errormessage(cid string, action string, err error) {
	m.State.message = err.Error()
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
	}
	app := &Dry{}
	newHeader(state)
	app.State = state
	app.header = newHeader(state)
	app.dockerDaemon = mdocker.ConnectToDaemon()
	app.screen = screen
	app.dockerDaemonRenderer = ui.NewDockerRenderer(
		app.dockerDaemon,
		screen.Cursor,
		state.SortMode,
		app.header)
	return app
}

//Not sure if this is the right approach
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
