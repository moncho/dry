package app

// Per-view key handling, extracted mechanically from handleKeyPress in
// model.go. Bodies are unchanged; behavior is locked by the golden view
// snapshots and the key-handling tests.

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/appui"
)

// handleContainersKeys handles key presses for the Main (containers) view.
func (m model) handleContainersKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if c := m.containers.SelectedContainer(); c != nil {
			m.containerMenu = appui.NewContainerMenuModel(c)
			m.containerMenu.SetSize(m.width, m.height)
			m.overlay = overlayContainerMenu
			return m, nil
		}
		return m, nil
	case "l", "L":
		if c := m.containers.SelectedContainer(); c != nil {
			return m, showContainerLogsCmd(m.daemon, c.ID)
		}
		return m, nil
	case "s":
		if c := m.containers.SelectedContainer(); c != nil {
			return m, showContainerStatsCmd(m.daemon, c.ID)
		}
		return m, nil
	case "e":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.showPrompt(
				fmt.Sprintf("Remove container %s?", shortID(c.ID)),
				"rm", c.ID,
			), nil
		}
		return m, nil
	case "x":
		if c := m.containers.SelectedContainer(); c != nil {
			var cmd tea.Cmd
			m.inputPrompt, cmd = appui.NewInputPromptModelWithLimit(
				fmt.Sprintf("Exec in %s:", shortID(c.ID)),
				"/bin/sh", "exec", c.ID, 120,
			)
			m.inputPrompt.SetSize(m.width, m.height)
			m.overlay = overlayInputPrompt
			return m, cmd
		}
		return m, nil
	case "ctrl+e":
		return m.showPrompt(
			"Remove all stopped containers?",
			"rm-all-stopped", "",
		), nil
	case "ctrl+k":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.showPrompt(
				fmt.Sprintf("Kill container %s?", shortID(c.ID)),
				"kill", c.ID,
			), nil
		}
		return m, nil
	case "ctrl+r":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.showPrompt(
				fmt.Sprintf("Restart container %s?", shortID(c.ID)),
				"restart", c.ID,
			), nil
		}
		return m, nil
	case "ctrl+t":
		if c := m.containers.SelectedContainer(); c != nil {
			return m.showPrompt(
				fmt.Sprintf("Stop container %s?", shortID(c.ID)),
				"stop", c.ID,
			), nil
		}
		return m, nil
	case "f1":
		// Change sort mode — need to reload with new sort
		var cmd tea.Cmd
		m.containers, cmd = m.containers.Update(msg)
		if m.daemon != nil {
			return m, tea.Batch(cmd,
				loadContainersCmd(m.daemon, m.containers.ShowAll(), m.containers.SortMode()))
		}
		return m, cmd
	case "f2":
		// Toggle show all — need to reload after
		var cmd tea.Cmd
		m.containers, cmd = m.containers.Update(msg)
		if m.daemon != nil {
			return m, tea.Batch(cmd,
				loadContainersCmd(m.daemon, m.containers.ShowAll(), m.containers.SortMode()))
		}
		return m, cmd
	case "f5":
		// Refresh
		if m.daemon != nil {
			return m, loadContainersCmd(m.daemon, m.containers.ShowAll(), m.containers.SortMode())
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.containers, cmd = m.containers.Update(msg)
	return m, cmd

}
