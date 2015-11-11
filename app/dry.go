package app

import (
	"fmt"
	"io"

	mdocker "github.com/moncho/dry/docker"
)

const help = `
<white>dry</>

Connects to a Docker daemon if environment variable DOCKER_HOST (and DOCKER_TLS_VERIFY, and DOCKER_CERT_PATH) is present
then shows the list of containers and allows some interaction with them.

<u>Command</u>    <u>Description                                </u>
	<white>F1</>       Cycles through containers sort modes (by Id | by Image | by Status | by Name)
	<white>F2</>       Toggles showing all containers (default shows just running)
	<white>F5</>       Refresh container list
	<white>k</>        Kills the selected container
	<white>l</>        Fetch the logs of the selected container
	<white>r</>        Restarts selected container (noop if it is already running)
	<white>s</>        Displays a live stream of the selected container resource usage statistics
	<white>t</>        Stops selected container (noop if it is not running)
	<white>q</>        Quits mop.
	<white>esc</>      Ditto.
<r> Press any key to continue </r>
`
const KeyMappings = "H<b><white>elp</></b> Q<b><white>uit</></b> <blue>|</> " +
	"F1:<b><white>Sort</></b> F2:<b><white>Toggle Show Containers</></b> F5:<b><white>Refresh</></b> <blue>|</> " +
	"<b><white>R</></b>e<b><white>move</></b> K<b><white>ill</></b> L<b><white>ogs</></b> R<b><white>estart</></b> S<b><white>tats</></b> <b><white>S</></b>t<b><white>op</></b>"

//Dry is the application representation.
type Dry struct {
	State        *AppState
	dockerDaemon *mdocker.DockerDaemon
	stats        *mdocker.Stats
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

func (m *Dry) Render() interface{} {
	if m.State.ShowingHelp {
		return help
	}
	if m.State.showingStats && m.stats != nil {
		return m.stats
	}
	if m.State.changed {
		m.dockerDaemon.Refresh(m.State.showingAllContainers)
	}
	m.State.changed = false
	return m.dockerDaemon
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
func NewDryApp() *Dry {
	state := &AppState{
		changed:              false,
		message:              "",
		Paused:               false,
		showingAllContainers: false,
		ShowingHelp:          false,
		SortMode:             mdocker.SortByContainerID,
	}
	app := &Dry{}
	app.State = state
	app.dockerDaemon = mdocker.ConnectToDaemon()
	return app
}
