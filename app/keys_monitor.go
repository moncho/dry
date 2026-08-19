package app

// Per-view key handling, extracted mechanically from handleKeyPress in
// model.go. Bodies are unchanged; behavior is locked by the golden view
// snapshots and the key-handling tests.

import (
	tea "charm.land/bubbletea/v2"
)

// handleMonitorKeys handles key presses for the Monitor view.
func (m model) handleMonitorKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.monitor, cmd = m.monitor.Update(msg)
	return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

}
