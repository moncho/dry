package app

// Per-view key handling, extracted mechanically from handleKeyPress in
// model.go. Bodies are unchanged; behavior is locked by the golden view
// snapshots and the key-handling tests.

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// handleVolumesKeys handles key presses for the Volumes view.
func (m model) handleVolumesKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if v := m.volumes.SelectedVolume(); v != nil {
			return m, inspectVolumeCmd(m.daemon, v.Name)
		}
		return m, nil
	case "ctrl+a":
		return m.showPrompt("Remove all volumes?", "vol-rm-all", ""), nil
	case "ctrl+e":
		if v := m.volumes.SelectedVolume(); v != nil {
			return m.showPrompt(
				fmt.Sprintf("Remove volume %s?", v.Name),
				"vol-rm", v.Name,
			), nil
		}
		return m, nil
	case "ctrl+f":
		if v := m.volumes.SelectedVolume(); v != nil {
			return m.showPrompt(
				fmt.Sprintf("Force remove volume %s?", v.Name),
				"vol-rm-force", v.Name,
			), nil
		}
		return m, nil
	case "ctrl+u":
		return m.showPrompt("Remove unused volumes?", "vol-prune", ""), nil
	case "f5":
		return m, loadVolumesCmd(m.daemon)
	}
	var cmd tea.Cmd
	m.volumes, cmd = m.volumes.Update(msg)
	return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

}
