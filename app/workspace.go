package app

// Workspace mode: pane layout, focus, pinning, and activity wiring.
// Moved verbatim from model.go.

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/moncho/dry/appui"
)

type workspacePane int

const (
	workspacePaneNavigator workspacePane = iota
	workspacePaneContext
	workspacePaneActivity
)

func (m model) renderWorkspaceFooter(bindings []key.Binding, footerBg, keyStyle, descStyle, sepStyle lipgloss.Style) string {
	renderCompact := func(bindings []key.Binding, mode int) string {
		var parts []string
		for _, kb := range bindings {
			if !kb.Enabled() {
				continue
			}
			keyText := kb.Help().Key
			if !m.swarmMode && (keyText == "5" || keyText == "6" || keyText == "7") {
				continue
			}
			descText := kb.Help().Desc
			switch mode {
			case 1:
				switch keyText {
				case "tab", "↑↓", "↵", "F1", "F2", "F5":
					descText = ""
				}
			case 2:
				descText = ""
			}
			if descText == "" {
				parts = append(parts, keyStyle.Render(keyText))
				continue
			}
			parts = append(parts, keyStyle.Render(keyText)+footerBg.Render(" ")+descStyle.Render(descText))
		}
		return strings.Join(parts, sepStyle.Render(" · "))
	}

	right := renderCompact([]key.Binding{key.NewBinding(key.WithKeys("h"), key.WithHelp("?", ""))}, 0)
	rightWidth := ansi.StringWidth(right)
	maxLeftWidth := m.width - rightWidth - 1
	if maxLeftWidth < 0 {
		maxLeftWidth = 0
	}

	left := renderCompact(bindings, 0)
	leftWidth := ansi.StringWidth(left)
	if leftWidth > maxLeftWidth {
		left = renderCompact(bindings, 1)
		leftWidth = ansi.StringWidth(left)
	}
	if leftWidth > maxLeftWidth {
		left = renderCompact(bindings, 2)
		leftWidth = ansi.StringWidth(left)
	}
	if leftWidth > maxLeftWidth {
		left = ansi.Truncate(left, maxLeftWidth, "")
		leftWidth = ansi.StringWidth(left)
	}
	line := left
	if leftWidth < m.width-rightWidth {
		line += footerBg.Render(strings.Repeat(" ", m.width-leftWidth-rightWidth))
	}
	line += right
	return appui.PadLine(line, m.width, footerBg)
}

func (m model) workspaceViewFooterBindings() []key.Binding {
	var enter []key.Binding
	var functionKeys []key.Binding
	for _, binding := range m.viewFooterBindings() {
		switch binding.Help().Key {
		case "F1", "F2", "F5":
			functionKeys = append(functionKeys, workspaceFunctionBinding(binding))
		case "enter":
			enter = append(enter, key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "open")))
		}
	}
	return append(enter, functionKeys...)
}

func workspaceFunctionBinding(binding key.Binding) key.Binding {
	switch binding.Help().Key {
	case "F1":
		return key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "sort"))
	case "F2":
		return key.NewBinding(key.WithKeys("f2"), key.WithHelp("F2", "all"))
	case "F5":
		return key.NewBinding(key.WithKeys("f5"), key.WithHelp("F5", "ref"))
	default:
		return binding
	}
}

func (m model) workspaceEnabled() bool {
	return m.config.WorkspaceMode
}

func (m model) workspaceCompactMode() bool {
	return m.width < minWorkspaceW || m.contentHeight() < minWorkspaceH
}

func (m model) workspaceLayout() (navigatorW, contextW, topH, activityH int) {
	usableH := m.contentHeight() - 1 // reserve one line for workspace tabs
	if usableH < 1 {
		return m.width, 0, 0, 0
	}
	if m.workspaceCompactMode() || usableH < minActivityH+minTopH {
		return m.width, 0, usableH, 0
	}

	// Base activity pane height, then derive topH.
	activityH = defaultActivityH
	if usableH < defaultActivityH+containerTopPaneCap+1 {
		activityH = compactActivityH
	}
	topH = usableH - activityH
	if topH < minTopH {
		return m.width, 0, usableH, 0
	}

	// Per-view cap: shrink top pane, give remainder to activity.
	topH = min(topH, m.topPaneCap())
	activityH = usableH - topH

	// Horizontal split for navigator / context panes.
	navigatorW = max(1, min(m.width*navigatorPct/100, m.width-minContextW))
	if navigatorW < minNavigatorW && m.width >= minNavigatorW+minContextW {
		navigatorW = minNavigatorW
	}
	contextW = max(1, m.width-navigatorW)
	return
}

func (m model) renderWorkspaceBody() string {
	_, _, topH, activityH := m.workspaceLayout()
	tabs := m.renderWorkspaceTabs()
	if topH <= 0 {
		return tabs
	}

	if m.workspaceCompactMode() {
		m.workspaceLogs.SetFocused(m.activePane == workspacePaneActivity)
		if m.activePane == workspacePaneActivity {
			return lipgloss.JoinVertical(lipgloss.Left, tabs, m.workspaceLogs.View())
		}
		return lipgloss.JoinVertical(lipgloss.Left, tabs, m.renderCurrentView())
	}

	m.populateWorkspaceContextPane()
	m.workspaceContext.SetFocused(m.activePane == workspacePaneContext)
	m.workspaceLogs.SetFocused(m.activePane == workspacePaneActivity)

	top := lipgloss.JoinHorizontal(lipgloss.Top, m.renderCurrentView(), m.workspaceContext.View())
	if activityH <= 0 {
		return lipgloss.JoinVertical(lipgloss.Left, tabs, top)
	}
	activity := m.workspaceLogs.View()
	return lipgloss.JoinVertical(lipgloss.Left, tabs, top, activity)
}

func (m *model) populateWorkspaceContextPane() {
	context := m.pinnedContext
	if context == nil {
		// Preview mode renders the full lines, so this needs the complete
		// context; only the pinned branch below can use the cheap target.
		if current, ok := m.currentWorkspacePreview(); ok {
			context = &current
		}
	}
	if context != nil {
		m.workspaceContext.SetEmptyMessage("")
		if m.pinnedContext != nil {
			mode := "pinned"
			// While pinned the preview no longer follows the cursor. Surface
			// what the cursor is on (and that p re-pins) in the pane header,
			// which never scrolls; putting it in the subtitle would reset the
			// body's scroll position on every cursor move.
			if current, ok := m.currentWorkspacePreviewTarget(); ok && current.identity() != context.identity() {
				mode = "pinned · cursor: " + current.title + " (p re-pins)"
			}
			m.workspaceContext.SetMode(mode)
		} else {
			m.workspaceContext.SetMode("preview")
		}
		m.workspaceContext.SetContent(context.title, context.subtitle, context.lines)
		return
	}
	m.workspaceContext.SetEmptyMessage(m.workspaceContextEmptyMessage())
	m.workspaceContext.SetMode("empty")
	m.workspaceContext.SetContent("", "", nil)
}

func (m model) workspaceContextEmptyMessage() string {
	switch m.view {
	case Main:
		return "Select a container to preview it here."
	case Images:
		return "Select an image to preview it here."
	case Networks:
		return "Select a network to preview it here."
	case Volumes:
		return "Select a volume to preview it here."
	case Monitor:
		return "Select a monitor row to preview live stats here."
	case Nodes:
		return "Select a node to preview it here."
	case Services:
		return "Select a service to preview it here."
	case Stacks:
		return "Select a stack to preview it here."
	case Tasks, ServiceTasks, StackTasks:
		return "Select a task to preview it here."
	case ComposeProjects:
		return "Select a Compose project or service to preview it here."
	case ComposeServices:
		return "Select a Compose resource to preview it here."
	default:
		return "Select an item to preview it here."
	}
}

func (m model) renderWorkspaceTabs() string {
	bg := lipgloss.NewStyle().Background(appui.DryTheme.Footer)
	active := lipgloss.NewStyle().
		Foreground(appui.DryTheme.Bg).
		Background(appui.DryTheme.Info).
		Bold(true).
		Padding(0, 1)
	inactive := lipgloss.NewStyle().
		Foreground(appui.DryTheme.FgMuted).
		Background(appui.DryTheme.Footer).
		Padding(0, 1)
	sep := lipgloss.NewStyle().
		Foreground(appui.DryTheme.FgSubtle).
		Background(appui.DryTheme.Footer).
		Render(" ")

	tab := func(label string, pane workspacePane) string {
		if m.activePane == pane {
			return active.Render(label)
		}
		return inactive.Render(label)
	}

	line := strings.Join([]string{
		tab("Navigator", workspacePaneNavigator),
		tab("Context", workspacePaneContext),
		tab("Activity", workspacePaneActivity),
	}, sep)
	if m.workspaceCompactMode() {
		line = strings.Join([]string{
			tab("Navigator", workspacePaneNavigator),
			tab("Activity", workspacePaneActivity),
		}, sep)
	}
	return appui.PadLine(line, m.width, bg)
}

func (m model) nextWorkspacePane(reverse bool) workspacePane {
	if m.workspaceCompactMode() {
		if m.activePane == workspacePaneActivity {
			return workspacePaneNavigator
		}
		return workspacePaneActivity
	}
	order := []workspacePane{
		workspacePaneNavigator,
		workspacePaneContext,
		workspacePaneActivity,
	}
	idx := slices.Index(order, m.activePane)
	if idx < 0 {
		idx = 0
	}
	if reverse {
		idx = (idx - 1 + len(order)) % len(order)
	} else {
		idx = (idx + 1) % len(order)
	}
	return order[idx]
}

func (m model) currentWorkspaceSelection() (workspaceContext, bool) {
	switch m.view {
	case Main:
		if c := m.containers.SelectedContainer(); c != nil {
			return workspaceContextFromContainer(c), true
		}
	case ComposeProjects:
		if svc := m.composeProjects.SelectedService(); svc != nil {
			return workspaceContextFromComposeService(*svc), true
		}
		if p := m.composeProjects.SelectedProject(); p != nil {
			return workspaceContextFromComposeProject(*p), true
		}
	case ComposeServices:
		if svc := m.composeServices.SelectedService(); svc != nil {
			return workspaceContextFromComposeService(*svc), true
		}
	}
	return workspaceContext{}, false
}

func (m model) currentWorkspacePreview() (workspaceContext, bool) {
	return m.workspacePreview(true)
}

// currentWorkspacePreviewTarget is a cheap variant used on every render while
// pinned, and by the command palette, where only the identity and title are
// needed. For the Monitor view (the hot path: one render per stats message)
// it builds no stat lines, copies no series, and does no daemon lookup; the
// other kinds share the full builders, which are cheap and render rarely.
func (m model) currentWorkspacePreviewTarget() (workspaceContext, bool) {
	return m.workspacePreview(false)
}

func (m model) workspacePreview(full bool) (workspaceContext, bool) {
	if ctx, ok := m.currentWorkspaceSelection(); ok {
		return ctx, true
	}
	switch m.view {
	case Images:
		if img := m.images.SelectedImage(); img != nil {
			return workspaceContextFromImage(*img), true
		}
	case Networks:
		if n := m.networks.SelectedNetwork(); n != nil {
			return workspaceContextFromNetwork(*n), true
		}
	case Volumes:
		if v := m.volumes.SelectedVolume(); v != nil {
			return workspaceContextFromVolume(v), true
		}
	case Monitor:
		if s := m.monitor.SelectedStats(); s != nil {
			if !full {
				return workspaceMonitorTarget(s), true
			}
			return workspaceContextFromStats(s, m.daemon, m.monitor.SelectedSeries()), true
		}
	case Nodes:
		if n := m.nodes.SelectedNode(); n != nil {
			return workspaceContextFromNode(*n), true
		}
	case Services:
		if s := m.services.SelectedService(); s != nil {
			return workspaceContextFromSwarmService(*s), true
		}
	case Stacks:
		if s := m.stacks.SelectedStack(); s != nil {
			return workspaceContextFromStack(*s), true
		}
	case Tasks, ServiceTasks, StackTasks:
		if t := m.tasks.SelectedTask(); t != nil {
			return workspaceContextFromTask(*t), true
		}
	}
	if m.view == ComposeServices {
		if p, ok := m.findComposeProjectByName(m.selectedProject); ok {
			return workspaceContextFromComposeProject(*p), true
		}
	}
	return workspaceContext{}, false
}

func (m model) toggleWorkspacePin() (tea.Model, tea.Cmd) {
	ctx, ok := m.currentWorkspacePreview()
	if m.pinnedContext != nil {
		// Pin already set: unpin when the cursor is still on the pinned item,
		// otherwise move the pin to follow the cursor's current selection.
		if !ok || ctx.identity() == m.pinnedContext.identity() {
			cleared := m.clearPinnedContext()
			return cleared, cleared.workspaceSelectionActivityCmd()
		}
	} else if !ok {
		return m, nil
	}
	return m.pinWorkspaceContext(ctx)
}

// pinWorkspacePreview pins the current cursor preview unconditionally.
// Unlike toggleWorkspacePin it never unpins, so a palette "Pin Current
// Preview" action cannot flip into an unpin when the cursor state changed
// between palette open and execution.
func (m model) pinWorkspacePreview() (tea.Model, tea.Cmd) {
	ctx, ok := m.currentWorkspacePreview()
	if !ok {
		return m, nil
	}
	return m.pinWorkspaceContext(ctx)
}

func (m model) pinWorkspaceContext(ctx workspaceContext) (tea.Model, tea.Cmd) {
	m.pinnedContext = &ctx
	m.workspaceLogs.SetContent("Activity", "Locking activity to pinned context", "Preparing pinned view...")
	m.closeActivityReader()
	if m.daemon == nil {
		return m, nil
	}
	return m, loadWorkspaceActivityCmd(m.daemon, ctx, m.workspaceLogs.Width(), m.workspaceLogs.BodyHeight())
}

func (m *model) clearPinnedContext() model {
	m.pinnedContext = nil
	m.closeActivityReader()
	title, status, content := m.workspaceActivityResetState()
	m.workspaceLogs.Clear(title, status, content)
	return *m
}

func (m *model) resetWorkspaceActivity() {
	m.closeActivityReader()
	title, status, content := m.workspaceActivityResetState()
	m.workspaceLogs.Clear(title, status, content)
}

func (m model) workspaceActivityResetState() (title, status, content string) {
	switch m.view {
	case Main, ComposeProjects, ComposeServices:
		return "Activity", "Idle · logs follow pinned selection", "Pin a container or Compose resource to stream logs here."
	case Images:
		return "Image Inspect", "Waiting for image selection", "Select an image to inspect it here."
	case Monitor:
		return "Monitor Details", "Waiting for monitor selection", "Select a monitor row to inspect live stats here."
	case Networks:
		return "Network Inspect", "Waiting for network selection", "Select a network to inspect it here."
	case Nodes:
		return "Node Inspect", "Waiting for node selection", "Select a node to inspect it here."
	case Services:
		return "Service Inspect", "Waiting for service selection", "Select a service to inspect it here."
	case Stacks:
		return "Stack Details", "Waiting for stack selection", "Select a stack to inspect its related resources here."
	case Tasks, ServiceTasks, StackTasks:
		return "Task Inspect", "Waiting for task selection", "Select a task to inspect it here."
	case Volumes:
		return "Volume Inspect", "Waiting for volume selection", "Select a volume to inspect it here."
	default:
		return "Activity", "Idle", "Select an item to populate activity here."
	}
}

func (m model) workspaceSelectionActivityCmd() tea.Cmd {
	if !m.workspaceEnabled() || m.daemon == nil || m.pinnedContext != nil {
		return nil
	}
	switch m.view {
	case Images:
		if img := m.images.SelectedImage(); img != nil {
			return loadWorkspaceImageInspectCmd(m.daemon, img.ID)
		}
	case Monitor:
		if stats := m.monitor.SelectedStats(); stats != nil {
			return loadWorkspaceMonitorDetails(m.daemon, stats, m.monitor.SelectedSeries(), m.workspaceLogs.Width(), m.workspaceLogs.BodyHeight())
		}
	case Networks:
		if n := m.networks.SelectedNetwork(); n != nil {
			return loadWorkspaceNetworkInspectCmd(m.daemon, n.ID)
		}
	case Nodes:
		if n := m.nodes.SelectedNode(); n != nil {
			return loadWorkspaceNodeInspectCmd(m.daemon, n.ID)
		}
	case Services:
		if s := m.services.SelectedService(); s != nil {
			return loadWorkspaceServiceInspectCmd(m.daemon, s.ID)
		}
	case Stacks:
		if s := m.stacks.SelectedStack(); s != nil {
			return loadWorkspaceStackDetailsCmd(m.daemon, *s)
		}
	case Tasks, ServiceTasks, StackTasks:
		if t := m.tasks.SelectedTask(); t != nil {
			return loadWorkspaceTaskInspectCmd(m.daemon, t.ID)
		}
	case Volumes:
		if v := m.volumes.SelectedVolume(); v != nil {
			return loadWorkspaceVolumeInspectCmd(m.daemon, v.Name)
		}
	default:
		return nil
	}
	return func() tea.Msg {
		title, status, content := m.workspaceActivityResetState()
		return workspaceActivityLoadedMsg{
			title:   title,
			status:  status,
			content: content,
		}
	}
}

func (m *model) refreshPinnedWorkspaceContext() {
	if m.pinnedContext == nil || m.daemon == nil {
		return
	}
	switch m.pinnedContext.kind {
	case workspaceContextContainer:
		if c, ok := m.findContainerByID(m.pinnedContext.containerID); ok {
			ctx := workspaceContextFromContainer(c)
			m.pinnedContext = &ctx
		}
	case workspaceContextComposeProject:
		if p, ok := m.findComposeProjectByName(m.pinnedContext.project); ok {
			ctx := workspaceContextFromComposeProject(*p)
			m.pinnedContext = &ctx
		}
	case workspaceContextComposeService:
		if svc, ok := m.findComposeService(m.pinnedContext.project, m.pinnedContext.service); ok {
			ctx := workspaceContextFromComposeService(*svc)
			m.pinnedContext = &ctx
		}
	case workspaceContextMonitor:
		if stats := m.monitor.StatsByID(m.pinnedContext.monitorCID); stats != nil {
			ctx := workspaceContextFromStats(stats, m.daemon, m.monitor.SeriesFor(m.pinnedContext.monitorCID))
			m.pinnedContext = &ctx
		}
	}
}

func (m model) workspaceMonitorActivityCmdThrottled() tea.Cmd {
	if !m.workspaceEnabled() || m.daemon == nil {
		return nil
	}
	if m.pinnedContext != nil {
		if m.pinnedContext.kind == workspaceContextMonitor {
			return loadWorkspaceActivityCmd(m.daemon, *m.pinnedContext, m.workspaceLogs.Width(), m.workspaceLogs.BodyHeight())
		}
		return nil
	}
	return m.workspaceSelectionActivityCmd()
}

func (m model) workspaceMonitorActivityCmd(cid string) tea.Cmd {
	if !m.workspaceEnabled() || m.daemon == nil {
		return nil
	}
	if m.pinnedContext != nil {
		if m.pinnedContext.kind == workspaceContextMonitor && m.pinnedContext.monitorCID == cid {
			return loadWorkspaceActivityCmd(m.daemon, *m.pinnedContext, m.workspaceLogs.Width(), m.workspaceLogs.BodyHeight())
		}
		return nil
	}
	return m.workspaceSelectionActivityCmd()
}
