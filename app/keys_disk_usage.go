package app

// Per-view key handling, extracted mechanically from handleKeyPress in
// model.go. Bodies are unchanged; behavior is locked by the golden view
// snapshots and the key-handling tests.

import (
	tea "charm.land/bubbletea/v2"
)

// handleDiskUsageKeys handles key presses for the Disk Usage view.
func (m model) handleDiskUsageKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "p", "P":
		return m.showPrompt("Prune all unused Docker resources?", "prune", ""), nil
	case "f5":
		return m, loadDiskUsageCmd(m.daemon)
	}
	var cmd tea.Cmd
	m.diskUsage, cmd = m.diskUsage.Update(msg)
	return m, cmd

}
