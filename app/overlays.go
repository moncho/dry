package app

// Overlay lifecycle: the overlay kinds, their key routing, and the prompt and quick peek entry points.
// Moved verbatim from model.go.

import (
	tea "charm.land/bubbletea/v2"
	"github.com/moncho/dry/appui"
)

type overlayType int

const (
	overlayNone overlayType = iota
	overlayLess
	overlayPrompt
	overlayInputPrompt
	overlayContainerMenu
	overlayCommandPalette
	overlayQuickPeek
)

func (m model) handleOverlayKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.overlay {
	case overlayLess:
		var cmd tea.Cmd
		m.less, cmd = m.less.Update(msg)
		return m, cmd
	case overlayPrompt:
		var cmd tea.Cmd
		m.prompt, cmd = m.prompt.Update(msg)
		return m, cmd
	case overlayInputPrompt:
		var cmd tea.Cmd
		m.inputPrompt, cmd = m.inputPrompt.Update(msg)
		return m, cmd
	case overlayContainerMenu:
		var cmd tea.Cmd
		m.containerMenu, cmd = m.containerMenu.Update(msg)
		return m, cmd
	case overlayCommandPalette:
		var cmd tea.Cmd
		m.commandPalette, cmd = m.commandPalette.Update(msg)
		return m, cmd
	case overlayQuickPeek:
		var cmd tea.Cmd
		m.quickPeek, cmd = m.quickPeek.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) openQuickPeek() (tea.Model, tea.Cmd) {
	ctx, ok := m.currentWorkspacePreview()
	if !ok {
		return m, nil
	}
	m.quickPeek = appui.NewQuickPeekModel()
	m.quickPeek.SetSize(m.width, m.height)
	m.quickPeek.SetContent(
		ctx.title,
		ctx.subtitle,
		"Preview",
		"Loading preview...",
		ctx.lines,
		"Preparing quick peek...",
	)
	m.overlay = overlayQuickPeek
	if m.daemon == nil {
		return m, nil
	}
	return m, loadQuickPeekCmd(m.daemon, ctx)
}

func (m model) showPrompt(message, tag, id string) model {
	m.prompt = appui.NewPromptModel(message, tag, id)
	m.prompt.SetWidth(m.width)
	m.overlay = overlayPrompt
	return m
}
