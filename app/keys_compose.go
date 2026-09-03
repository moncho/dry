package app

// Per-view key handling, originally extracted from handleKeyPress in
// model.go and since changed here: behaviour is locked by the golden view
// snapshots and the key-handling tests.

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
)

// handleComposeProjectsKeys handles key presses for the Compose projects view.
func (m model) handleComposeProjectsKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Service row: inspect the service's first container
		if svc := m.composeProjects.SelectedService(); svc != nil {
			return m, inspectComposeServiceCmd(m.daemon, svc.Project, svc.Name)
		}
		// Project row: drill into project resources
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m.openComposeServices(p.Name)
		}
		return m, m.composeNoProjectCmd("Inspect")
	case "l", "L":
		if svc := m.composeProjects.SelectedService(); svc != nil {
			return m, showComposeLogsCmd(m.daemon, svc.Project, svc.Name)
		}
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m, showComposeLogsCmd(m.daemon, p.Name, "")
		}
		return m, m.composeNoProjectCmd("Logs")
	case "f5":
		return m, loadComposeProjectsCmd(m.daemon)
	case "ctrl+t":
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m.showPrompt(fmt.Sprintf("Stop project %s?", p.Name),
				"compose-project-stop", p.Name), nil
		}
	case "ctrl+r":
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m.showPrompt(fmt.Sprintf("Restart project %s?", p.Name),
				"compose-project-restart", p.Name), nil
		}
	case "ctrl+e":
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m.showPrompt(fmt.Sprintf("Remove project %s containers?", p.Name),
				"compose-project-rm", p.Name), nil
		}
	case "u":
		if svc := m.composeProjects.SelectedService(); svc != nil {
			if p := m.composeProjects.ProjectByName(svc.Project); p != nil {
				return m, composeUpCmd(m.composeCLI, *p, svc.Name)
			}
		}
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m, composeUpCmd(m.composeCLI, *p)
		}
		return m, m.composeNoProjectCmd("Up")
	case "d":
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

// composeSelectionCmd reports that a key has no row it can act on. The
// Compose Services view mixes networks and volumes in with its services, so
// several documented keys land on rows they cannot act on, and a key that
// does nothing and says nothing reads as broken.
func composeSelectionCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return statusMessageMsg{text: text, expiry: 3 * time.Second}
	}
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
		if svc := m.composeServices.SelectedService(); svc != nil {
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
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m, showComposeLogsCmd(m.daemon, svc.Project, svc.Name)
		}
		return m, m.composeNotAServiceCmd("Logs")
	case "f5":
		reload := m.loadComposeServices(m.selectedProject)
		return m, reload
	case "ctrl+s":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Start service %s?", svc.Name),
				"compose-start", svc.Project+"/"+svc.Name), nil
		}
		return m, m.composeNotAServiceCmd("Start")
	case "ctrl+t":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Stop service %s?", svc.Name),
				"compose-stop", svc.Project+"/"+svc.Name), nil
		}
		return m, m.composeNotAServiceCmd("Stop")
	case "ctrl+r":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Restart service %s?", svc.Name),
				"compose-restart", svc.Project+"/"+svc.Name), nil
		}
		return m, m.composeNotAServiceCmd("Restart")
	case "ctrl+e":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Remove service %s containers?", svc.Name),
				"compose-rm", svc.Project+"/"+svc.Name), nil
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
