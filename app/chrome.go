package app

// Screen chrome and layout: main-screen composition, footer, loading screen, theme rotation, and content sizing.
// Moved verbatim from model.go.

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/version"
)

func (m model) renderMainScreen() string {
	return m.renderMainScreenWithFooter(m.renderFooter())
}

// renderMainScreenWithFooter renders the main screen with the given
// bottom line (footer, prompt, or input prompt).
func (m model) renderMainScreenWithFooter(footer string) string {
	var sections []string

	if m.showHeader {
		sections = append(sections, m.header.View())
		sections = append(sections, m.header.SeparatorLine(m.messageBar.Message()))
	}

	if m.workspaceEnabled() {
		sections = append(sections, m.renderWorkspaceBody())
	} else {
		switch m.view {
		case Main:
			sections = append(sections, m.containers.View())
		case Images:
			sections = append(sections, m.images.View())
		case Networks:
			sections = append(sections, m.networks.View())
		case Volumes:
			sections = append(sections, m.volumes.View())
		case DiskUsage:
			sections = append(sections, m.diskUsage.View())
		case Monitor:
			sections = append(sections, m.monitor.View())
		case Nodes:
			sections = append(sections, m.nodes.View())
		case Services:
			sections = append(sections, m.services.View())
		case Stacks:
			sections = append(sections, m.stacks.View())
		case ServiceTasks, Tasks, StackTasks:
			sections = append(sections, m.tasks.View())
		case ComposeProjects:
			sections = append(sections, m.composeProjects.View())
		case ComposeServices:
			sections = append(sections, m.composeServices.View())
		default:
			sections = append(sections, "View not yet implemented")
		}
	}

	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m model) renderFooter() string {
	footerBg := lipgloss.NewStyle().Background(appui.DryTheme.Footer)
	keyStyle := lipgloss.NewStyle().Foreground(appui.DryTheme.Key).Background(appui.DryTheme.Footer)
	descStyle := lipgloss.NewStyle().Foreground(appui.DryTheme.FgSubtle).Background(appui.DryTheme.Footer)
	sepStyle := lipgloss.NewStyle().Foreground(appui.DryTheme.FgSubtle).Background(appui.DryTheme.Footer)

	if m.workspaceEnabled() {
		leftBindings := []key.Binding{
			key.NewBinding(key.WithKeys("1-8", "m"), key.WithHelp("1-8/m", "nav")),
			key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab/⇧tab", "pane")),
			key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pin")),
			key.NewBinding(key.WithKeys("space"), key.WithHelp("spc", "peek")),
			key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "move")),
		}
		leftBindings = append(leftBindings, m.workspaceViewFooterBindings()...)
		return m.renderWorkspaceFooter(leftBindings, footerBg, keyStyle, descStyle, sepStyle)
	}

	var bindings []key.Binding
	switch m.view {
	case Main:
		bindings = containerKeys.ShortHelp()
	case Monitor:
		bindings = monitorKeys.ShortHelp()
	case Images:
		bindings = imagesKeys.ShortHelp()
	case Networks:
		bindings = networksKeys.ShortHelp()
	case Volumes:
		bindings = volumesKeys.ShortHelp()
	case DiskUsage:
		bindings = diskUsageKeys.ShortHelp()
	case Services:
		bindings = servicesKeys.ShortHelp()
	case Stacks:
		bindings = stacksKeys.ShortHelp()
	case Nodes:
		bindings = nodesKeys.ShortHelp()
	case ServiceTasks, Tasks, StackTasks:
		bindings = tasksKeys.ShortHelp()
	case ComposeProjects:
		bindings = composeProjectsKeys.ShortHelp()
	case ComposeServices:
		bindings = composeServicesKeys.ShortHelp()
	default:
		bindings = containerKeys.ShortHelp()
	}
	bindings = append(bindings, globalKeys.Theme)
	bindings = append(bindings, globalKeys.Palette)
	bindings = append(bindings, globalKeys.QuickPeek)

	// Whole bindings, never half of one: the strip is cut to the terminal
	// width, and cutting inside an entry leaves "^e rm stop", which reads
	// as a rendering fault where a list ending in an ellipsis reads as a
	// list that did not fit (dry's own "^e rm stopped" is what that looks
	// like). The marker is added only when it fits in what is left over,
	// since dropping a whole binding to make room for it would say less
	// than the binding did.
	const marker = "  \u2026"
	renderBindings := func(bindings []key.Binding, limit int) (line string, width int, dropped bool) {
		var b strings.Builder
		first := true
		for _, kb := range bindings {
			if !kb.Enabled() {
				continue
			}
			// Hide swarm navigation keys when swarm is not active.
			if !m.swarmMode {
				k := kb.Help().Key
				if k == "5" || k == "6" || k == "7" {
					continue
				}
			}
			sep := ""
			if !first {
				sep = "  \u00b7  "
			}
			entry := kb.Help().Key + " " + kb.Help().Desc
			if width+ansi.StringWidth(sep)+ansi.StringWidth(entry) > limit {
				// Whether anything was emitted is what first records; the
				// loop index counts the bindings skipped above as well.
				return b.String(), width, !first
			}
			if sep != "" {
				b.WriteString(sepStyle.Render(sep))
			}
			width += ansi.StringWidth(sep) + ansi.StringWidth(entry)
			first = false
			b.WriteString(keyStyle.Render(kb.Help().Key))
			b.WriteString(footerBg.Render(" "))
			b.WriteString(descStyle.Render(kb.Help().Desc))
		}
		return b.String(), width, false
	}

	// A zero or negative width needs no special case: the first binding
	// does not fit, so nothing is emitted and the padding below has
	// nothing to pad.
	line, width, dropped := renderBindings(bindings, m.width)
	if dropped && width+ansi.StringWidth(marker) <= m.width {
		line += sepStyle.Render(marker)
	}
	if w := ansi.StringWidth(line); w < m.width {
		line += footerBg.Render(strings.Repeat(" ", m.width-w))
	}
	return line
}

func (m model) viewFooterBindings() []key.Binding {
	switch m.view {
	case Main:
		return containerKeys.ShortHelp()
	case Monitor:
		return monitorKeys.ShortHelp()
	case Images:
		return imagesKeys.ShortHelp()
	case Networks:
		return networksKeys.ShortHelp()
	case Volumes:
		return volumesKeys.ShortHelp()
	case DiskUsage:
		return diskUsageKeys.ShortHelp()
	case Services:
		return servicesKeys.ShortHelp()
	case Stacks:
		return stacksKeys.ShortHelp()
	case Nodes:
		return nodesKeys.ShortHelp()
	case ServiceTasks, Tasks, StackTasks:
		return tasksKeys.ShortHelp()
	case ComposeProjects:
		return composeProjectsKeys.ShortHelp()
	case ComposeServices:
		return composeServicesKeys.ShortHelp()
	default:
		return containerKeys.ShortHelp()
	}
}

func (m *model) rotateTheme() {
	appui.RotateColorTheme()
	appui.InitStyles()
	m.containers.RefreshTableStyles()
	m.images.RefreshTableStyles()
	m.networks.RefreshTableStyles()
	m.volumes.RefreshTableStyles()
	m.monitor.RefreshTableStyles()
	m.nodes.RefreshTableStyles()
	m.services.RefreshTableStyles()
	m.stacks.RefreshTableStyles()
	m.tasks.RefreshTableStyles()
	m.composeProjects.RefreshTableStyles()
	m.composeServices.RefreshTableStyles()
}

// topPaneCap returns the maximum top pane height for the current view.
func (m model) topPaneCap() int {
	switch m.view {
	case Main:
		return containerTopPaneCap
	case Monitor:
		return monitorFramingLines + max(1, min(m.monitor.RowCount(), maxMonitorRows))
	default:
		return int(^uint(0) >> 1) // max int — no cap
	}
}

func (m *model) resizeContentModels() {
	ch := m.contentHeight()
	width := m.width
	height := ch
	m.containers.SetCompact(m.workspaceEnabled())
	if m.workspaceEnabled() {
		width, _, height, _ = m.workspaceLayout()
	}

	m.containers.SetSize(width, height)
	m.images.SetSize(width, height)
	m.networks.SetSize(width, height)
	m.volumes.SetSize(width, height)
	m.diskUsage.SetSize(width, height)
	m.monitor.SetSize(width, height)
	m.nodes.SetSize(width, height)
	m.services.SetSize(width, height)
	m.stacks.SetSize(width, height)
	m.tasks.SetSize(width, height)
	m.composeProjects.SetSize(width, height)
	m.composeServices.SetSize(width, height)

	if m.workspaceEnabled() {
		_, contextW, topH, activityH := m.workspaceLayout()
		if m.workspaceCompactMode() {
			if m.activePane == workspacePaneContext {
				m.activePane = workspacePaneNavigator
			}
			m.workspaceContext.SetSize(0, 0)
			m.workspaceLogs.SetSize(m.width, topH)
		} else {
			m.workspaceContext.SetSize(contextW, topH)
			m.workspaceLogs.SetSize(m.width, activityH)
		}
	}
}

func (m model) renderCurrentView() string {
	switch m.view {
	case Main:
		return m.containers.View()
	case Images:
		return m.images.View()
	case Networks:
		return m.networks.View()
	case Volumes:
		return m.volumes.View()
	case DiskUsage:
		return m.diskUsage.View()
	case Monitor:
		return m.monitor.View()
	case Nodes:
		return m.nodes.View()
	case Services:
		return m.services.View()
	case Stacks:
		return m.stacks.View()
	case ServiceTasks, Tasks, StackTasks:
		return m.tasks.View()
	case ComposeProjects:
		return m.composeProjects.View()
	case ComposeServices:
		return m.composeServices.View()
	default:
		return "View not yet implemented"
	}
}

func (m model) renderLoadingScreen() string {
	frames := []string{
		docker.Whale0, docker.Whale1, docker.Whale2, docker.Whale3,
		docker.Whale4, docker.Whale5, docker.Whale6, docker.Whale7, docker.Whale,
	}

	frame := ""
	if m.loadingFrame < len(frames) {
		frame = frames[m.loadingFrame]
	}

	connecting := "\U0001f433 Trying to connect to the Docker Host \U0001f433"

	// Top line: connecting message, centered
	connectLine := ui.White(connecting)
	topLine := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, connectLine)

	// Middle: whale, centered
	whale := ui.Cyan(frame)
	whaleBlock := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, whale)

	// Fill the middle area so the whale is vertically centered.
	// Account for: 1 top line, 2 bottom lines, whale height.
	whaleHeight := strings.Count(whale, "\n") + 1
	bottomLines := 2
	padding := m.height - 1 - whaleHeight - bottomLines
	topPad := padding / 2
	botPad := padding - topPad
	if topPad < 0 {
		topPad = 0
	}
	if botPad < 0 {
		botPad = 0
	}

	// Bottom-left: version + host
	verLine := ui.Blue("Dry Version: ") + ui.White(version.VERSION)
	hostLine := ""
	if m.config.DockerHost != "" {
		hostLine = ui.Blue("Docker Host: ") + ui.White(m.config.DockerHost)
	}

	// Bottom-right: attribution
	attribution := "made with \U0001f499 (and go) by moncho"

	// Compose bottom two lines
	bottomRow1 := verLine
	if m.width > 0 {
		attrW := ansi.StringWidth(attribution)
		verW := ansi.StringWidth(verLine)
		gap := m.width - verW - attrW
		if gap > 0 {
			bottomRow1 = verLine + strings.Repeat(" ", gap) + attribution
		}
	}
	bottomRow2 := hostLine

	var sections []string
	sections = append(sections, topLine)
	if topPad > 0 {
		sections = append(sections, strings.Repeat("\n", topPad-1))
	}
	sections = append(sections, whaleBlock)
	if botPad > 0 {
		sections = append(sections, strings.Repeat("\n", botPad-1))
	}
	sections = append(sections, bottomRow1)
	if bottomRow2 != "" {
		sections = append(sections, bottomRow2)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *model) advanceLoadingFrame() {
	maxFrame := 8
	if m.loadingFwd {
		m.loadingFrame++
		if m.loadingFrame >= maxFrame {
			m.loadingFwd = false
		}
	} else {
		m.loadingFrame--
		if m.loadingFrame <= 0 {
			m.loadingFwd = true
		}
	}
}

func (m model) contentHeight() int {
	h := m.height
	if m.showHeader {
		h -= appui.MainScreenHeaderSize // 3 info lines
		h--                             // separator line
	}
	h -= appui.MainScreenFooterLength
	return h
}
