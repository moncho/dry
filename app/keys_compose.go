package app

// Per-view key handling, originally extracted from handleKeyPress in
// model.go and since changed here: behaviour is locked by the golden view
// snapshots and the key-handling tests.

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/moncho/dry/docker"
)

// handleComposeProjectsKeys handles key presses for the Compose projects view.
func (m model) handleComposeProjectsKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Service row: inspect the service's first container
		svc, why := containerAction(m.composeProjects.SelectedService(), "Inspect")
		if why != nil {
			return m, why
		}
		if svc != nil {
			return m, inspectComposeServiceCmd(m.daemon, svc.Project, svc.Name)
		}
		// Project row: drill into project resources
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m.openComposeServices(p.Name)
		}
		return m, m.composeNoProjectCmd("Inspect")
	case "l", "L":
		svc, why := containerAction(m.composeProjects.SelectedService(), "Logs")
		if why != nil {
			return m, why
		}
		if svc != nil {
			return m, showComposeLogsCmd(m.daemon, svc.Project, svc.Name)
		}
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m, showComposeLogsCmd(m.daemon, p.Name, "")
		}
		return m, m.composeNoProjectCmd("Logs")
	case "f5":
		return m, loadComposeProjectsCmd(m.daemon)
	case "ctrl+t":
		return m.composeProjectsLifecycle("Stop", "compose-stop", "compose-project-stop",
			"Stop service %s?", "Stop project %s?")
	case "ctrl+r":
		return m.composeProjectsLifecycle("Restart", "compose-restart", "compose-project-restart",
			"Restart service %s?", "Restart project %s?")
	case "ctrl+e":
		return m.composeProjectsLifecycle("Remove", "compose-rm", "compose-project-rm",
			"Remove service %s containers?", "Remove project %s containers?")
	case "u":
		if svc := m.composeProjects.SelectedService(); svc != nil {
			if p := m.composeProjects.ProjectByName(svc.Project); p != nil {
				return m, composeUpCmd(m.composeCLI, *p, svc.Name)
			}
		}
		if p := m.composeProjects.SelectedProject(); p != nil {
			// A whole project asks first, like every other project-level
			// action here. It used to run unasked, which made the cursor
			// load-bearing: any refresh that moved the selection onto a
			// header turned "bring up this service" into "bring up
			// everything", and no amount of cursor-following covers a
			// filter, a sort, or a row type nobody has written yet.
			return m.showPrompt(fmt.Sprintf("Bring project %s up?", p.Name),
				"compose-project-up", p.Name), nil
		}
		return m, m.composeNoProjectCmd("Up")
	case "d":
		// The one lifecycle key with no per-service form: compose down
		// removes the project's networks along with every container. On a
		// service row it says where it lives rather than taking the parent
		// project down, which is what the three keys above used to do.
		if svc := m.composeProjects.SelectedService(); svc != nil {
			// The project's row, not "the row above": that is only the
			// header for a project's first service. And a filter can be
			// hiding that row, so the message says so rather than sending
			// the reader to a row that is not on screen.
			if m.composeProjects.Filtered() {
				return m, composeSelectionCmd(fmt.Sprintf(
					"Down takes a whole project, not the service %s: clear the filter to reach %s",
					svc.Name, svc.Project))
			}
			return m, composeSelectionCmd(fmt.Sprintf(
				"Down takes a whole project, not the service %s: press d on the %s row",
				svc.Name, svc.Project))
		}
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m.showPrompt(fmt.Sprintf("Take project %s down?", p.Name),
				"compose-project-down", p.Name), nil
		}
		return m, m.composeNoProjectCmd("Down")
	case "c":
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m, composeConfigCmd(m.composeCLI, *p)
		}
		return m, m.composeNoProjectCmd("Config")
	}
	var cmd tea.Cmd
	m.composeProjects, cmd = m.composeProjects.Update(msg)
	return m, cmd

}

// composeProjectsLifecycle points a lifecycle key at the row under the
// cursor: the selected service, or the project when a header is selected.
// All three keys took SelectedProject() before, which returns the parent
// project for a service row, so ctrl+e on a service row showing no
// containers prompted "Remove project web containers?" and removed every
// container in the project. enter, l and u had always acted per service, so
// the same cursor meant two different targets depending on the key.
//
// The service prompts reuse the Compose Services view's tags, which is
// where the same three keys already act per service.
func (m model) composeProjectsLifecycle(action, serviceTag, projectTag, servicePrompt, projectPrompt string) (tea.Model, tea.Cmd) {
	if selected := m.composeProjects.SelectedService(); selected != nil {
		svc, why := containerAction(selected, action)
		if why != nil {
			return m, why
		}
		// Qualified by project: this view lists every project at once, so
		// "Stop service web?" names two different services on a host with
		// two projects that both have one.
		return m.showPrompt(fmt.Sprintf(servicePrompt, svc.Project+"/"+svc.Name),
			serviceTag, composeServiceID(svc.Project, svc.Name)), nil
	}
	if p := m.composeProjects.SelectedProject(); p != nil {
		return m.showPrompt(fmt.Sprintf(projectPrompt, p.Name), projectTag, p.Name), nil
	}
	return m, m.composeNoProjectCmd(action)
}

// composeNoServiceRowCmd says an action needs a service row in the Compose
// Projects view, where a project can be selected and have none: a project
// with no containers and nothing in its compose file has a header and
// nothing under it.
func (m model) composeNoServiceRowCmd(action string) tea.Cmd {
	if m.composeProjects.Filtered() {
		return composeSelectionCmd(action + " needs a service, and the filter matches none")
	}
	return composeSelectionCmd(action + " needs a service, and none is selected")
}

// composeSelectionCmd reports that a key has no row it can act on. The
// Compose Services view mixes networks and volumes in with its services, so
// several documented keys land on rows they cannot act on, and a key that
// does nothing and says nothing reads as broken.
func composeSelectionCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return statusMessageMsg{text: text, expiry: 3 * time.Second}
	}
}

// composeNoContainersCmd explains that an action on a service's containers
// was asked of a service that has none, which the Compose views now list.
// The service is named unqualified even in the projects view, whose prompts
// are qualified: a prompt has to be unambiguous because it is a yes to
// something destructive, where this names a key to press about the row the
// cursor is visibly on.
// The lifecycle actions used to prompt and then report "0 targeted, 0
// succeeded", so the user confirmed a destructive-sounding no-op; logs and
// inspect answered with their own wording and said nothing about the key
// that does apply. u is that key, so name it.
func composeNoContainersCmd(action, service string) tea.Cmd {
	return composeSelectionCmd(fmt.Sprintf(
		"%s needs a container, and %s has none: u brings it up", action, service))
}

// containerAction resolves the service an action should act on, or the
// command that says why it cannot. Every key and palette entry that needs
// a container to act on goes through here, eighteen call sites; written out
// per site instead, the palette's projects-view entries went without it. The
// exception on purpose is compose:recreate, which is up --force-recreate
// and so creates the service it is pointed at.
//
// A nil service with a nil command means no service is selected at all,
// which each caller words for itself.
func containerAction(svc *docker.ComposeService, action string) (*docker.ComposeService, tea.Cmd) {
	if svc == nil || svc.Containers > 0 {
		return svc, nil
	}
	return nil, composeNoContainersCmd(action, svc.Name)
}

// composeNoProjectCmd explains that a project key was pressed with no row
// under the cursor, which the Compose Projects view reaches whenever the
// list is empty: nothing loaded yet, a filter matching nothing, or a host
// with no compose projects. Same rule as the services view: a documented
// key that does nothing and says nothing reads as broken.
func (m model) composeNoProjectCmd(action string) tea.Cmd {
	switch {
	case m.composeProjects.Filtered():
		return composeSelectionCmd(action + " needs a project, and the filter matches none")
	case m.composeProjects.ProjectCount() == 0:
		return composeSelectionCmd(action + " needs a project, and none is loaded")
	default:
		return composeSelectionCmd(action + " needs a project, and none is selected")
	}
}

// composeNotAServiceCmd explains that a service key was pressed on
// something else. It names the row when there is one, because a highlighted
// network or volume plus "select a service first" reads as a bug in dry,
// and otherwise names the reason there is none: still loading, filtered
// down to nothing, or a project with no resources at all.
func (m model) composeNotAServiceCmd(action string) tea.Cmd {
	switch {
	case m.composeServices.SelectedNetwork() != nil:
		return composeSelectionCmd(action + " only applies to services, and a network is selected")
	case m.composeServices.SelectedVolume() != nil:
		return composeSelectionCmd(action + " only applies to services, and a volume is selected")
	case m.composeServices.Loading():
		// The load is several round trips behind the view switch, so
		// calling the project empty here would be a guess.
		return composeSelectionCmd(action + " needs a service, and this project is still loading")
	case m.composeServices.Filtered():
		// A filter can narrow the list to a header, the one row the cursor
		// rests on that resolves to nothing. Name the filter, not the row.
		return composeSelectionCmd(action + " only applies to services, and the filter matches no service")
	default:
		return composeSelectionCmd(action + " needs a service, and this project has none")
	}
}

// handleComposeServicesKeys handles key presses for the Compose services view.
func (m model) handleComposeServicesKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.workspaceEnabled() && m.pinnedContext != nil {
			cleared := m.clearPinnedContext()
			return cleared, cleared.workspaceSelectionActivityCmd()
		}
		m.view = ComposeProjects
		return m, loadComposeProjectsCmd(m.daemon)
	case "enter":
		svc, why := containerAction(m.composeServices.SelectedService(), "Inspect")
		if why != nil {
			return m, why
		}
		if svc != nil {
			return m, inspectComposeServiceCmd(m.daemon, svc.Project, svc.Name)
		}
		if n := m.composeServices.SelectedNetwork(); n != nil {
			return m, inspectNetworkCmd(m.daemon, n.Name)
		}
		if v := m.composeServices.SelectedVolume(); v != nil {
			return m, inspectVolumeCmd(m.daemon, v.Name)
		}
		switch {
		case m.composeServices.Loading():
			return m, composeSelectionCmd("Nothing to inspect yet: this project is still loading")
		case m.composeServices.Filtered():
			return m, composeSelectionCmd("Nothing to inspect: the filter matches no service, network or volume")
		default:
			return m, composeSelectionCmd("Nothing to inspect: this project has no services, networks or volumes")
		}
	case "l", "L":
		svc, why := containerAction(m.composeServices.SelectedService(), "Logs")
		if why != nil {
			return m, why
		}
		if svc != nil {
			return m, showComposeLogsCmd(m.daemon, svc.Project, svc.Name)
		}
		return m, m.composeNotAServiceCmd("Logs")
	case "f5":
		reload := m.loadComposeServices(m.selectedProject)
		return m, reload
	case "ctrl+s":
		svc, why := containerAction(m.composeServices.SelectedService(), "Start")
		if why != nil {
			return m, why
		}
		if svc != nil {
			return m.showPrompt(fmt.Sprintf("Start service %s?", svc.Name),
				"compose-start", composeServiceID(svc.Project, svc.Name)), nil
		}
		return m, m.composeNotAServiceCmd("Start")
	case "ctrl+t":
		svc, why := containerAction(m.composeServices.SelectedService(), "Stop")
		if why != nil {
			return m, why
		}
		if svc != nil {
			return m.showPrompt(fmt.Sprintf("Stop service %s?", svc.Name),
				"compose-stop", composeServiceID(svc.Project, svc.Name)), nil
		}
		return m, m.composeNotAServiceCmd("Stop")
	case "ctrl+r":
		svc, why := containerAction(m.composeServices.SelectedService(), "Restart")
		if why != nil {
			return m, why
		}
		if svc != nil {
			return m.showPrompt(fmt.Sprintf("Restart service %s?", svc.Name),
				"compose-restart", composeServiceID(svc.Project, svc.Name)), nil
		}
		return m, m.composeNotAServiceCmd("Restart")
	case "ctrl+e":
		svc, why := containerAction(m.composeServices.SelectedService(), "Remove")
		if why != nil {
			return m, why
		}
		if svc != nil {
			return m.showPrompt(fmt.Sprintf("Remove service %s containers?", svc.Name),
				"compose-rm", composeServiceID(svc.Project, svc.Name)), nil
		}
		return m, m.composeNotAServiceCmd("Remove")
	case "u":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m, composeUpCmd(m.composeCLI, m.composeProjectFor(svc.Project), svc.Name)
		}
		// Say where the whole-project version lives. Not "esc, then u":
		// in workspace mode with a pinned context the first esc only
		// clears the pin.
		if m.composeServices.SelectedNetwork() != nil || m.composeServices.SelectedVolume() != nil {
			return m, composeSelectionCmd("Up applies to a service here; back on the projects list, u brings up the whole project")
		}
		return m, m.composeNotAServiceCmd("Up")
	case "c":
		if m.selectedProject == "" {
			return m, composeSelectionCmd("Config needs a project, and none is open")
		}
		return m, composeConfigCmd(m.composeCLI, m.composeProjectFor(m.selectedProject))
	}
	var cmd tea.Cmd
	m.composeServices, cmd = m.composeServices.Update(msg)
	return m, cmd

}

// composeProjectsServiceOp is the palette's half of the projects view's
// lifecycle actions: the same gate, the same prompts and the same tags the
// keys use, so the two surfaces cannot drift into acting on different rows.
func (m model) composeProjectsServiceOp(action, tag, prompt string) (tea.Model, tea.Cmd) {
	svc, why := containerAction(m.composeProjects.SelectedService(), action)
	if why != nil {
		return m, why
	}
	if svc == nil {
		// Reached when the selection moved while the palette was open. The
		// projects view's own no-project wording would be wrong here: a
		// project can be selected and still have no service row under it.
		return m, m.composeNoServiceRowCmd(action)
	}
	return m.showPrompt(fmt.Sprintf(prompt, svc.Project+"/"+svc.Name),
		tag, composeServiceID(svc.Project, svc.Name)), nil
}
