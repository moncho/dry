package app

// Per-view key handling, originally extracted from handleKeyPress in
// model.go: behaviour is locked by the golden view snapshots and the
// key-handling tests.

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/appui"
)

// handleNodesKeys handles key presses for the Nodes view.
func (m model) handleNodesKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.swarmMode {
		switch msg.String() {
		case "enter":
			if n := m.nodes.SelectedNode(); n != nil {
				m.previousView = m.view
				m.view = Tasks
				return m, loadNodeTasksCmd(m.daemon, n.ID)
			}
			return m, nil
		case "i", "I":
			if n := m.nodes.SelectedNode(); n != nil {
				return m, inspectNodeCmd(m.daemon, n.ID)
			}
			return m, nil
		case "ctrl+a":
			if n := m.nodes.SelectedNode(); n != nil {
				return m, m.cycleNodeAvailability(n.ID)
			}
			return m, nil
		case "f5":
			return m, loadNodesCmd(m.daemon)
		}
	}
	var cmd tea.Cmd
	m.nodes, cmd = m.nodes.Update(msg)
	return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

}

// handleServicesKeys handles key presses for the Services view.
func (m model) handleServicesKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.swarmMode {
		switch msg.String() {
		case "enter":
			if s := m.services.SelectedService(); s != nil {
				m.previousView = m.view
				m.view = ServiceTasks
				return m, loadServiceTasksCmd(m.daemon, s.ID)
			}
			return m, nil
		case "i", "I":
			if s := m.services.SelectedService(); s != nil {
				return m, inspectServiceCmd(m.daemon, s.ID)
			}
			return m, nil
		case "l", "L":
			if s := m.services.SelectedService(); s != nil {
				return m, showServiceLogsCmd(m.daemon, s.ID)
			}
			return m, nil
		case "ctrl+r":
			if s := m.services.SelectedService(); s != nil {
				return m.showPrompt(
					fmt.Sprintf("Remove service %s?", s.Spec.Name),
					"service-rm", s.ID,
				), nil
			}
			return m, nil
		case "ctrl+s":
			if s := m.services.SelectedService(); s != nil {
				var cmd tea.Cmd
				m.inputPrompt, cmd = appui.NewInputPromptModel(
					fmt.Sprintf("Scale service %s to replicas:", s.Spec.Name),
					"number", "service-scale", s.ID,
				)
				m.inputPrompt.SetSize(m.width, m.height)
				m.overlay = overlayInputPrompt
				return m, cmd
			}
			return m, nil
		case "ctrl+u":
			if s := m.services.SelectedService(); s != nil {
				return m.showPrompt(
					fmt.Sprintf("Force update service %s?", s.Spec.Name),
					"service-update", s.ID,
				), nil
			}
			return m, nil
		case "f5":
			return m, loadServicesCmd(m.daemon)
		}
	}
	var cmd tea.Cmd
	m.services, cmd = m.services.Update(msg)
	return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

}

// handleStacksKeys handles key presses for the Stacks view.
func (m model) handleStacksKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.swarmMode {
		switch msg.String() {
		case "enter":
			if s := m.stacks.SelectedStack(); s != nil {
				m.previousView = m.view
				m.view = StackTasks
				return m, loadStackTasksCmd(m.daemon, s.Name)
			}
			return m, nil
		case "ctrl+r":
			if s := m.stacks.SelectedStack(); s != nil {
				return m.showPrompt(
					fmt.Sprintf("Remove stack %s?", s.Name),
					"stack-rm", s.Name,
				), nil
			}
			return m, nil
		case "f5":
			return m, loadStacksCmd(m.daemon)
		}
	}
	var cmd tea.Cmd
	m.stacks, cmd = m.stacks.Update(msg)
	return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

}

// handleTasksKeys handles key presses for the Tasks views.
func (m model) handleTasksKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = m.previousView
		reload := m.loadViewData(m.view)
		return m, reload
	}
	var cmd tea.Cmd
	m.tasks, cmd = m.tasks.Update(msg)
	return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())
}
