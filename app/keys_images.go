package app

// Per-view key handling, extracted mechanically from handleKeyPress in
// model.go. Bodies are unchanged; behavior is locked by the golden view
// snapshots and the key-handling tests.

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/docker"
)

// handleImagesKeys handles key presses for the Images view.
func (m model) handleImagesKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if img := m.images.SelectedImage(); img != nil {
			return m, inspectImageCmd(m.daemon, img.ID)
		}
		return m, nil
	case "i", "I":
		if img := m.images.SelectedImage(); img != nil {
			return m, showImageHistoryCmd(m.daemon, img.ID)
		}
		return m, nil
	case "ctrl+d":
		return m.showPrompt("Remove dangling images?", "rmi-dangling", ""), nil
	case "ctrl+e":
		if img := m.images.SelectedImage(); img != nil {
			return m.showPrompt(
				fmt.Sprintf("Remove image %s?", docker.TruncateID(docker.ImageID(img.ID))),
				"rmi", img.ID,
			), nil
		}
		return m, nil
	case "ctrl+f":
		if img := m.images.SelectedImage(); img != nil {
			return m.showPrompt(
				fmt.Sprintf("Force remove image %s?", docker.TruncateID(docker.ImageID(img.ID))),
				"rmi-force", img.ID,
			), nil
		}
		return m, nil
	case "ctrl+u":
		return m.showPrompt("Remove unused images?", "rmi-unused", ""), nil
	case "f5":
		return m, loadImagesCmd(m.daemon)
	}
	var cmd tea.Cmd
	m.images, cmd = m.images.Update(msg)
	return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

}
