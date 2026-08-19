package app

// Per-view key handling, extracted mechanically from handleKeyPress in
// model.go. Bodies are unchanged; behavior is locked by the golden view
// snapshots and the key-handling tests.

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// handleNetworksKeys handles key presses for the Networks view.
func (m model) handleNetworksKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if n := m.networks.SelectedNetwork(); n != nil {
			return m, inspectNetworkCmd(m.daemon, n.ID)
		}
		return m, nil
	case "ctrl+e":
		if n := m.networks.SelectedNetwork(); n != nil {
			return m.showPrompt(
				fmt.Sprintf("Remove network %s?", n.Name),
				"net-rm", n.ID,
			), nil
		}
		return m, nil
	case "f5":
		return m, loadNetworksCmd(m.daemon)
	}
	var cmd tea.Cmd
	m.networks, cmd = m.networks.Update(msg)
	return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

}
