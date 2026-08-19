package app

// Per-view key handling, extracted mechanically from handleKeyPress in
// model.go. Bodies are unchanged; behavior is locked by the golden view
// snapshots and the key-handling tests.

import (
	"fmt"

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
			m.previousView = m.view
			m.view = ComposeServices
			m.selectedProject = p.Name
			return m, loadComposeServicesCmd(m.daemon, p.Name)
		}
		return m, nil
	case "l", "L":
		if svc := m.composeProjects.SelectedService(); svc != nil {
			return m, showComposeLogsCmd(m.daemon, svc.Project, svc.Name)
		}
		if p := m.composeProjects.SelectedProject(); p != nil {
			return m, showComposeLogsCmd(m.daemon, p.Name, "")
		}
		return m, nil
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
	}
	var cmd tea.Cmd
	m.composeProjects, cmd = m.composeProjects.Update(msg)
	return m, cmd

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
		return m, nil
	case "l", "L":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m, showComposeLogsCmd(m.daemon, svc.Project, svc.Name)
		}
		return m, nil
	case "f5":
		return m, loadComposeServicesCmd(m.daemon, m.selectedProject)
	case "ctrl+s":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Start service %s?", svc.Name),
				"compose-start", svc.Project+"/"+svc.Name), nil
		}
	case "ctrl+t":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Stop service %s?", svc.Name),
				"compose-stop", svc.Project+"/"+svc.Name), nil
		}
	case "ctrl+r":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Restart service %s?", svc.Name),
				"compose-restart", svc.Project+"/"+svc.Name), nil
		}
	case "ctrl+e":
		if svc := m.composeServices.SelectedService(); svc != nil {
			return m.showPrompt(fmt.Sprintf("Remove service %s containers?", svc.Name),
				"compose-rm", svc.Project+"/"+svc.Name), nil
		}
	}
	var cmd tea.Cmd
	m.composeServices, cmd = m.composeServices.Update(msg)
	return m, cmd

}
