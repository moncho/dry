package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moncho/dry/appui"
	appcompose "github.com/moncho/dry/appui/compose"
	appswarm "github.com/moncho/dry/appui/swarm"
	appworkspace "github.com/moncho/dry/appui/workspace"
	"github.com/moncho/dry/docker"
)

// Compile-time assertion: model implements tea.Model.
var _ tea.Model = model{}

// Workspace layout constants.
const (
	minWorkspaceW    = 100 // terminal width below which workspace uses compact mode
	minWorkspaceH    = 12  // content height below which workspace uses compact mode
	minTopH          = 3   // minimum top pane height to show split layout
	minActivityH     = 4   // minimum usable activity pane height
	defaultActivityH = 8   // activity pane height on normal terminals
	compactActivityH = 5   // activity pane height on shorter terminals
	navigatorPct     = 58  // navigator width as percentage of terminal width
	minNavigatorW    = 40  // minimum navigator pane width
	minContextW      = 24  // minimum context pane width

	// Top pane caps per view.
	containerTopPaneCap = 9 // 5 data rows + widget/table framing
	monitorFramingLines = 4 // widget header + summary + table header + blank
	maxMonitorRows      = 5 // max visible monitor rows in workspace top pane
)

type model struct {
	// State
	view         viewMode
	previousView viewMode
	width        int
	height       int
	showHeader   bool
	ready        bool

	// Docker
	daemon       dockerDaemon
	config       Config
	swarmMode    bool
	eventsChan   <-chan events.Message
	eventsCancel context.CancelFunc
	composeCLI   composeEngine
	workingDir   string

	// Sub-models
	containers       appui.ContainersModel
	images           appui.ImagesModel
	networks         appui.NetworksModel
	volumes          appui.VolumesModel
	diskUsage        appui.DiskUsageModel
	monitor          appui.MonitorModel
	nodes            appswarm.NodesModel
	services         appswarm.ServicesModel
	stacks           appswarm.StacksModel
	tasks            appswarm.TasksModel
	composeProjects  appcompose.ProjectsModel
	composeServices  appcompose.ServicesModel
	workspaceContext appworkspace.ContextModel
	workspaceLogs    appworkspace.ActivityModel
	activePane       workspacePane
	pinnedContext    *workspaceContext
	selectedProject  string
	header           appui.HeaderModel
	messageBar       appui.MessageBarModel

	// Overlay state
	overlay        overlayType
	less           appui.LessModel
	prompt         appui.PromptModel
	inputPrompt    appui.InputPromptModel
	containerMenu  appui.ContainerMenuModel
	commandPalette appui.CommandPaletteModel
	quickPeek      appui.QuickPeekModel
	streamReader   io.ReadCloser // active streaming reader (logs)
	activityReader io.ReadCloser
	eventsLive     bool // true when events less overlay is open

	// Docker event throttling
	pendingRefresh map[docker.SourceType]bool
	refreshTimer   bool

	// composeCycleInFlight is true from the moment a compose project reload
	// starts a scan/drift cycle until that cycle's composeDriftMsg lands. One
	// cycle costs `compose config --format json` plus one `compose config
	// --hash=*` per project with files, each 150-400ms of CPU in its own
	// subprocess, so a second cycle started before the first finishes just
	// piles overlapping batches on top of each other.
	composeCycleInFlight bool

	// composeRefreshPending records that a compose reload was skipped, so it
	// runs once when the reason goes away. Dropping those refreshes outright
	// would leave the view showing pre-`up` state: the container event that
	// turns a project's status to running arrives exactly while the cycle or
	// the streamed output is still busy.
	composeRefreshPending bool

	// Monitor stats workspace throttling
	monitorStatsTimer bool

	// Loading animation
	loadingFrame int
	loadingFwd   bool

	// Splash screen
	splashDone bool
}

// NewModel creates a new top-level model.
func NewModel(cfg Config) model {
	workingDir, _ := os.Getwd()
	return model{
		workingDir:       workingDir,
		config:           cfg,
		view:             Main,
		showHeader:       true,
		containers:       appui.NewContainersModel(),
		images:           appui.NewImagesModel(),
		networks:         appui.NewNetworksModel(),
		volumes:          appui.NewVolumesModel(),
		diskUsage:        appui.NewDiskUsageModel(),
		monitor:          appui.NewMonitorModel(),
		nodes:            appswarm.NewNodesModel(),
		services:         appswarm.NewServicesModel(),
		stacks:           appswarm.NewStacksModel(),
		tasks:            appswarm.NewTasksModel(),
		composeProjects:  appcompose.NewProjectsModel(),
		composeServices:  appcompose.NewServicesModel(),
		workspaceContext: appworkspace.NewContextModel(),
		workspaceLogs:    appworkspace.NewActivityModel(),
		pendingRefresh:   make(map[docker.SourceType]bool),
		loadingFwd:       true,
		splashDone:       cfg.SplashDuration <= 0,
	}
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		connectToDockerCmd(m.config),
		loadingTickCmd(),
	}
	if m.config.SplashDuration > 0 {
		cmds = append(cmds, tea.Tick(m.config.SplashDuration, func(time.Time) tea.Msg {
			return splashDoneMsg{}
		}))
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeContentModels()
		m.header.SetWidth(m.width)
		m.less.SetSize(m.width, m.height)
		m.prompt.SetWidth(m.width)
		m.inputPrompt.SetSize(m.width, m.height)
		m.containerMenu.SetSize(m.width, m.height)
		m.commandPalette.SetSize(m.width, m.height)
		m.quickPeek.SetSize(m.width, m.height)
		return m, nil

	case dockerConnectedMsg:
		m.daemon = msg.daemon
		m.ready = m.splashDone
		if info, err := m.daemon.Info(); err == nil {
			m.swarmMode = info.Swarm.LocalNodeState == swarm.LocalNodeStateActive
		}
		m.monitor.SetDaemon(m.daemon)
		m.tasks.SetDaemon(m.daemon)
		m.resizeContentModels()
		m.header = appui.NewHeaderModel(m.daemon, m.width)
		eventsCtx, eventsCancel := context.WithCancel(context.Background())
		eventsCh, err := m.daemon.Events(eventsCtx)
		if err != nil {
			eventsCancel()
			m.messageBar.SetMessage(fmt.Sprintf("Docker events error: %s", err), 5*time.Second)
			return m, tea.Batch(
				loadContainersCmd(m.daemon, m.containers.ShowAll(), m.containers.SortMode()),
				loadHeaderInfoCmd(m.daemon),
				detectComposeCmd(m.daemon.DockerEnv()),
			)
		}
		m.eventsChan = eventsCh
		m.eventsCancel = eventsCancel
		if m.config.MonitorMode {
			m2, cmd := m.switchView(Monitor)
			return m2, tea.Batch(cmd, listenDockerEvents(m.eventsChan), loadHeaderInfoCmd(m.daemon), detectComposeCmd(m.daemon.DockerEnv()))
		}
		return m, tea.Batch(
			loadContainersCmd(m.daemon, m.containers.ShowAll(), m.containers.SortMode()),
			listenDockerEvents(m.eventsChan),
			loadHeaderInfoCmd(m.daemon),
			detectComposeCmd(m.daemon.DockerEnv()),
		)

	case headerInfoMsg:
		m.header.SetDockerInfo(msg.info, msg.infoErr, msg.ver, msg.verErr)
		return m, nil

	case composeDetectedMsg:
		if msg.err == nil && msg.cli != nil {
			m.composeCLI = msg.cli
		}
		return m, nil

	case dockerErrorMsg:
		// Fatal error — can't connect to Docker
		m.messageBar.SetMessage(fmt.Sprintf("Error: %s", msg.err), 10*time.Second)
		return m, tea.Quit

	case splashDoneMsg:
		m.splashDone = true
		if m.daemon != nil {
			m.ready = true
		}
		return m, nil

	case containersLoadedMsg:
		m.containers.SetContainers(msg.containers)
		m.refreshPinnedWorkspaceContext()
		return m, nil

	case appui.ImagesLoadedMsg:
		m.images.SetImages(msg.Images)
		return m, m.workspaceSelectionActivityCmd()

	case appui.NetworksLoadedMsg:
		m.networks.SetNetworks(msg.Networks)
		return m, m.workspaceSelectionActivityCmd()

	case appui.VolumesLoadedMsg:
		m.volumes.SetVolumes(msg.Volumes)
		return m, m.workspaceSelectionActivityCmd()

	case appui.DiskUsageLoadedMsg:
		m.diskUsage.SetUsage(msg.Usage)
		return m, nil

	case appui.MonitorStatsMsg:
		prevCount := m.monitor.StatsCount()
		cmd := m.monitor.UpdateStats(msg.CID, msg.Stats, msg.StatsCh)
		newContainer := m.monitor.StatsCount() != prevCount
		if newContainer {
			m.monitor.FlushTable()
			if m.workspaceEnabled() {
				m.resizeContentModels()
			}
		}
		cmds := []tea.Cmd{cmd}
		if !m.monitorStatsTimer {
			m.monitorStatsTimer = true
			cmds = append(cmds, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
				return flushMonitorStatsMsg{}
			}))
		}
		return m, tea.Batch(cmds...)

	case appui.MonitorErrorMsg:
		prevRows := m.monitor.RowCount()
		m.monitor.RemoveContainer(msg.CID)
		if m.workspaceEnabled() && m.monitor.RowCount() != prevRows {
			m.resizeContentModels()
		}
		m.refreshPinnedWorkspaceContext()
		return m, m.workspaceMonitorActivityCmd(msg.CID)

	case appswarm.NodesLoadedMsg:
		m.nodes.SetNodes(msg.Nodes)
		m.refreshPinnedWorkspaceContext()
		return m, m.workspaceSelectionActivityCmd()

	case appswarm.ServicesLoadedMsg:
		m.services.SetServices(msg.Services)
		m.refreshPinnedWorkspaceContext()
		return m, m.workspaceSelectionActivityCmd()

	case appswarm.StacksLoadedMsg:
		m.stacks.SetStacks(msg.Stacks)
		m.refreshPinnedWorkspaceContext()
		return m, m.workspaceSelectionActivityCmd()

	case appswarm.TasksLoadedMsg:
		m.tasks.SetTasks(msg.Tasks, msg.Title)
		return m, m.workspaceSelectionActivityCmd()

	case appcompose.ProjectsLoadedMsg:
		m.composeProjects.SetProjects(msg.Projects)
		m.refreshPinnedWorkspaceContext()
		// A cycle is now in flight; flushRefreshMsg will not start another
		// until composeDriftMsg ends this one. Every path out of here leads
		// to a composeDriftMsg — composeDriftCmd returns one even with no
		// engine at all — so the flag cannot get stuck.
		m.composeCycleInFlight = true
		// composeScanCmd and composeDriftCmd must never run concurrently
		// against the same []docker.ProjectWithServices: composeScanCmd
		// mutates project fields in place (docker.MergeScannedProject),
		// while composeDriftCmd reads those same fields. Batching them
		// together is a data race. When a scan will run, defer drift until
		// composeProjectsMsg carries the scan's (possibly enriched) result;
		// when no scan will run, composeProjectsMsg never arrives, so drift
		// must be dispatched here instead.
		if resolver, ok := m.composeCLI.(composeResolver); ok {
			return m, composeScanCmd(resolver, m.workingDir, msg.Projects)
		}
		return m, composeDriftCmd(m.composeCLI, msg.Projects, m.daemon.Containers(nil, docker.NoSort))

	case composeProjectsMsg:
		m.composeProjects.SetProjects(msg.projects)
		return m, composeDriftCmd(m.composeCLI, msg.projects, m.daemon.Containers(nil, docker.NoSort))

	case composeDriftMsg:
		m.composeCycleInFlight = false
		m.composeProjects.SetDrift(msg.drift)
		m.composeServices.SetDrift(msg.drift)
		cmds := []tea.Cmd{m.drainPendingComposeRefresh()}
		if msg.err != nil {
			cmds = append(cmds, func() tea.Msg {
				return statusMessageMsg{
					text:   fmt.Sprintf("Compose drift check failed: %s", msg.err),
					expiry: 5 * time.Second,
				}
			})
		}
		return m, tea.Batch(cmds...)

	case appcompose.ServicesLoadedMsg:
		m.composeServices.SetServices(msg.Services, msg.Networks, msg.Volumes, msg.Project)
		m.refreshPinnedWorkspaceContext()
		return m, nil

	case workspaceActivityLoadedMsg:
		m.closeActivityReader()
		m.workspaceLogs.SetContent(msg.title, msg.status, msg.content)
		if msg.reader != nil {
			m.activityReader = msg.reader
			return m, readWorkspaceActivityCmd(msg.reader)
		}
		return m, nil

	case appendWorkspaceActivityMsg:
		m.workspaceLogs.AppendContent(msg.content)
		return m, readWorkspaceActivityCmd(msg.reader)

	case workspaceActivityClosedMsg:
		m.closeActivityReader()
		return m, nil

	case quickPeekLoadedMsg:
		m.quickPeek.SetContent(
			msg.title,
			msg.subtitle,
			msg.detailTitle,
			msg.status,
			msg.summary,
			msg.content,
		)
		return m, nil

	case eventsClosedMsg:
		// Events channel was closed (daemon restart, network error).
		// Try to re-establish the events listener after a short delay.
		m.messageBar.SetMessage("Docker events disconnected, reconnecting...", 3*time.Second)
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return reconnectEventsMsg{}
		})

	case reconnectEventsMsg:
		if m.daemon == nil {
			return m, nil
		}
		// Cancel the old event goroutines before creating new ones.
		if m.eventsCancel != nil {
			m.eventsCancel()
		}
		eventsCtx, eventsCancel := context.WithCancel(context.Background())
		eventsCh, err := m.daemon.Events(eventsCtx)
		if err != nil {
			eventsCancel()
			m.messageBar.SetMessage(fmt.Sprintf("Events reconnect failed: %s", err), 5*time.Second)
			return m, tea.Tick(5*time.Second, func(time.Time) tea.Msg {
				return reconnectEventsMsg{}
			})
		}
		m.eventsChan = eventsCh
		m.eventsCancel = eventsCancel
		m.messageBar.SetMessage("Docker events reconnected", 3*time.Second)
		return m, listenDockerEvents(m.eventsChan)

	case dockerEventMsg:
		if m.eventsLive && m.overlay == overlayLess {
			m.less.AppendContent(formatEvent(msg.event) + "\n")
		}
		source := docker.SourceType(msg.event.Type)
		m.pendingRefresh[source] = true
		cmds := []tea.Cmd{listenDockerEvents(m.eventsChan)}
		if !m.refreshTimer {
			m.refreshTimer = true
			cmds = append(cmds, tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
				return flushRefreshMsg{}
			}))
		}
		return m, tea.Batch(cmds...)

	case flushRefreshMsg:
		m.refreshTimer = false
		var cmds []tea.Cmd
		for source := range m.pendingRefresh {
			switch source {
			case docker.ContainerSource:
				if m.view == Main {
					cmds = append(cmds, loadContainersCmd(m.daemon, m.containers.ShowAll(), m.containers.SortMode()))
				}
				// A compose reload is not free: its ProjectsLoadedMsg starts
				// a scan/drift cycle of compose subprocesses. Container
				// events arrive fastest exactly when that cycle is most
				// expensive — during a streamed `up` — so skip the reload
				// while one cycle is still running, and skip it while a
				// streaming viewer covers the view the reload would repaint.
				if m.view == ComposeProjects {
					if m.composeCycleInFlight || m.streamingViewerOpen() {
						m.composeRefreshPending = true
					} else {
						cmds = append(cmds, loadComposeProjectsCmd(m.daemon))
					}
				}
				if m.view == ComposeServices {
					if m.streamingViewerOpen() {
						m.composeRefreshPending = true
					} else {
						cmds = append(cmds, loadComposeServicesCmd(m.daemon, m.selectedProject))
					}
				}
			case docker.ImageSource:
				if m.view == Images {
					cmds = append(cmds, loadImagesCmd(m.daemon))
				}
			case docker.NetworkSource:
				if m.view == Networks {
					cmds = append(cmds, loadNetworksCmd(m.daemon))
				}
			case docker.VolumeSource:
				if m.view == Volumes {
					cmds = append(cmds, loadVolumesCmd(m.daemon))
				}
			case docker.ServiceSource:
				if m.swarmMode && m.view == Services {
					cmds = append(cmds, loadServicesCmd(m.daemon))
				}
			case docker.NodeSource:
				if m.swarmMode && m.view == Nodes {
					cmds = append(cmds, loadNodesCmd(m.daemon))
				}
			}
		}
		m.pendingRefresh = make(map[docker.SourceType]bool)
		return m, tea.Batch(cmds...)

	case flushMonitorStatsMsg:
		m.monitorStatsTimer = false
		m.monitor.FlushTable()
		m.refreshPinnedWorkspaceContext()
		return m, m.workspaceMonitorActivityCmdThrottled()

	case operationSuccessMsg:
		m.messageBar.SetMessage(msg.message, 3*time.Second)
		return m, m.loadViewData(m.view)

	case operationErrorMsg:
		m.messageBar.SetMessage(fmt.Sprintf("Error: %s", msg.err), 5*time.Second)
		return m, nil

	case statusMessageMsg:
		m.messageBar.SetMessage(msg.text, msg.expiry)
		return m, tea.Tick(msg.expiry, func(time.Time) tea.Msg {
			return messageBarExpiredMsg{}
		})
	case execEndedMsg:
		m.messageBar.SetMessage(msg.text, msg.expiry)
		return m, tea.Batch(
			tea.ClearScreen,
			tea.RequestWindowSize,
			tea.Tick(msg.expiry, func(time.Time) tea.Msg {
				return messageBarExpiredMsg{}
			}),
		)
	case messageBarExpiredMsg:
		return m, nil

	case showLessMsg:
		m.less = appui.NewLessModel()
		m.less.SetSize(m.width, m.height)
		m.less.SetContent(msg.content, msg.title)
		m.overlay = overlayLess
		if msg.title == "Docker Events" {
			m.eventsLive = true
			m.less.SetFollowing(true)
		}
		return m, nil

	case showStreamingLessMsg:
		// Two streams can be dispatched before either message lands (key
		// repeat on a slow daemon); the superseded reader must be closed or
		// its follow-mode HTTP connection leaks until process exit.
		if m.streamReader != nil {
			_ = m.streamReader.Close()
		}
		m.less = appui.NewLessModel()
		m.less.SetSize(m.width, m.height)
		m.less.SetContent(msg.content, msg.title)
		m.less.SetFollowing(true)
		m.overlay = overlayLess
		m.streamReader = msg.reader
		return m, readLogStreamCmd(msg.reader)

	case appendLessMsg:
		// Only append content from the reader that is currently live;
		// a chunk from a superseded stream would interleave two containers'
		// logs in one viewer.
		if m.overlay == overlayLess && m.streamReader != nil && msg.reader == m.streamReader {
			m.less.AppendContent(msg.content)
			return m, readLogStreamCmd(msg.reader)
		}
		// Overlay was closed or the stream was replaced: clean up the reader
		if msg.reader != nil && msg.reader != m.streamReader {
			_ = msg.reader.Close()
		}
		return m, nil

	case streamClosedMsg:
		// A close notice from a superseded stream must not detach the live
		// one; that would leak it when the overlay closes. The same guard
		// keeps a stale stream's error from being reported after the user
		// has already moved on to a newer one.
		if msg.reader == nil || msg.reader == m.streamReader {
			m.streamReader = nil
			if msg.err != nil {
				return m, func() tea.Msg {
					return statusMessageMsg{
						text:   fmt.Sprintf("Command failed: %s", msg.err),
						expiry: 8 * time.Second,
					}
				}
			}
		}
		return m, nil

	case appui.CloseOverlayMsg:
		m.overlay = overlayNone
		m.eventsLive = false
		var cmds []tea.Cmd
		if m.streamReader != nil {
			err := m.streamReader.Close()
			m.streamReader = nil
			if err != nil {
				cmds = append(cmds, func() tea.Msg {
					return statusMessageMsg{
						text:   fmt.Sprintf("Command failed: %s", err),
						expiry: 8 * time.Second,
					}
				})
			}
		}
		// The refreshes skipped behind the stream are what kept the view
		// underneath current; run one now that it is visible again.
		cmds = append(cmds, m.drainPendingComposeRefresh())
		return m, tea.Batch(cmds...)

	case appui.PromptResultMsg:
		m.overlay = overlayNone
		if msg.Confirmed {
			return m, m.executeContainerOp(msg.Tag, msg.ID)
		}
		return m, nil

	case appui.InputPromptResultMsg:
		m.overlay = overlayNone
		if !msg.Cancelled {
			return m, m.executeInputOp(msg.Tag, msg.ID, msg.Value)
		}
		return m, nil

	case appui.ContainerMenuCommandMsg:
		m.overlay = overlayNone
		return m.executeMenuCommand(msg.ContainerID, msg.Command)

	case appui.CommandPaletteResultMsg:
		m.overlay = overlayNone
		return m.executePaletteAction(msg.ActionID)

	case loadingTickMsg:
		if m.ready {
			return m, nil
		}
		m.advanceLoadingFrame()
		return m, loadingTickCmd()

	case tea.KeyPressMsg:
		// When an overlay is active, forward keys to it
		if m.overlay != overlayNone {
			return m.handleOverlayKeyPress(msg)
		}
		return m.handleKeyPress(msg)

	case tea.MouseWheelMsg:
		if m.overlay == overlayLess {
			var cmd tea.Cmd
			m.less, cmd = m.less.Update(msg)
			return m, cmd
		}
		if m.workspaceEnabled() {
			switch m.activePane {
			case workspacePaneContext:
				m.populateWorkspaceContextPane()
				var cmd tea.Cmd
				m.workspaceContext, cmd = m.workspaceContext.Update(msg)
				return m, cmd
			case workspacePaneActivity:
				var cmd tea.Cmd
				m.workspaceLogs, cmd = m.workspaceLogs.Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Quit keys always handled regardless of filter state
	switch msg.String() {
	case "ctrl+c", "Q":
		m.monitor.StopAll()
		if m.streamReader != nil {
			// dry is exiting; there is nowhere left to show a failure, so the
			// close error is deliberately discarded here (unlike the
			// CloseOverlayMsg path above, which reports it).
			_ = m.streamReader.Close()
			m.streamReader = nil
		}
		if m.eventsCancel != nil {
			m.eventsCancel()
		}
		m.closeActivityReader()
		return m, tea.Quit
	}

	// When a sub-model's filter input is active, delegate directly
	// so keys like Esc, numbers, etc. go to the filter, not global handlers.
	if m.filterActive() {
		return m.delegateToSubModel(msg)
	}

	// Global keys
	switch msg.String() {
	case ":":
		return m.openCommandPalette()
	case " ", "space":
		return m.openQuickPeek()
	case "tab", "shift+tab", "backtab":
		if m.workspaceEnabled() {
			m.activePane = m.nextWorkspacePane(msg.String() == "shift+tab" || msg.String() == "backtab")
			return m, nil
		}
	case "P", "p":
		if m.workspaceEnabled() {
			return m.toggleWorkspacePin()
		}
	case "f7":
		m.showHeader = !m.showHeader
		m.resizeContentModels()
		return m, nil
	case "ctrl+0":
		m.rotateTheme()
		return m, nil
	case "1":
		return m.switchView(Main)
	case "?", "h", "H":
		return m, showHelpCmd()
	case "f9":
		if m.daemon != nil {
			return m, showDockerEventsCmd(m.daemon)
		}
		return m, nil
	case "f10":
		if m.daemon != nil {
			return m, showDockerInfoCmd(m.daemon)
		}
		return m, nil
	case "2":
		return m.switchView(Images)
	case "3":
		return m.switchView(Networks)
	case "4":
		return m.switchView(Volumes)
	case "m", "M":
		return m.switchView(Monitor)
	case "f8":
		return m.switchView(DiskUsage)
	case "5":
		if !m.swarmMode {
			return m, nil
		}
		return m.switchView(Nodes)
	case "6":
		if !m.swarmMode {
			return m, nil
		}
		return m.switchView(Services)
	case "7":
		if !m.swarmMode {
			return m, nil
		}
		return m.switchView(Stacks)
	case "8":
		return m.switchView(ComposeProjects)
	case "esc":
		if m.workspaceEnabled() && m.pinnedContext != nil {
			cleared := m.clearPinnedContext()
			return cleared, cleared.workspaceSelectionActivityCmd()
		}
		// Escape goes back to main from any non-main, non-task, non-compose-services view
		if m.view != Main && m.view != ServiceTasks && m.view != Tasks && m.view != StackTasks && m.view != ComposeServices {
			return m.switchView(Main)
		}
	}

	if m.workspaceEnabled() && m.activePane == workspacePaneActivity {
		var cmd tea.Cmd
		m.workspaceLogs, cmd = m.workspaceLogs.Update(msg)
		return m, cmd
	}
	if m.workspaceEnabled() && m.activePane == workspacePaneContext {
		m.populateWorkspaceContextPane()
		var cmd tea.Cmd
		m.workspaceContext, cmd = m.workspaceContext.Update(msg)
		return m, cmd
	}

	// Delegate to active sub-model
	switch m.view {
	case Main:
		return m.handleContainersKeys(msg)
	case Images:
		return m.handleImagesKeys(msg)
	case Networks:
		return m.handleNetworksKeys(msg)
	case Volumes:
		return m.handleVolumesKeys(msg)
	case DiskUsage:
		return m.handleDiskUsageKeys(msg)
	case Monitor:
		return m.handleMonitorKeys(msg)
	case Nodes:
		return m.handleNodesKeys(msg)
	case Services:
		return m.handleServicesKeys(msg)
	case Stacks:
		return m.handleStacksKeys(msg)
	case ComposeProjects:
		return m.handleComposeProjectsKeys(msg)
	case ComposeServices:
		return m.handleComposeServicesKeys(msg)
	case ServiceTasks, Tasks, StackTasks:
		return m.handleTasksKeys(msg)
	}
	return m, nil
}

// filterActive returns true if the current view's sub-model has an active filter input.
func (m model) filterActive() bool {
	switch m.view {
	case Main:
		return m.containers.FilterActive()
	case Images:
		return m.images.FilterActive()
	case Networks:
		return m.networks.FilterActive()
	case Volumes:
		return m.volumes.FilterActive()
	case Nodes:
		return m.nodes.FilterActive()
	case Services:
		return m.services.FilterActive()
	case Stacks:
		return m.stacks.FilterActive()
	case Tasks, ServiceTasks, StackTasks:
		return m.tasks.FilterActive()
	case ComposeProjects:
		return m.composeProjects.FilterActive()
	case ComposeServices:
		return m.composeServices.FilterActive()
	}
	return false
}

// delegateToSubModel forwards a key event directly to the active sub-model,
// bypassing global key handling. Used when a filter input is active.
func (m model) delegateToSubModel(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.view {
	case Main:
		m.containers, cmd = m.containers.Update(msg)
	case Images:
		m.images, cmd = m.images.Update(msg)
	case Networks:
		m.networks, cmd = m.networks.Update(msg)
	case Volumes:
		m.volumes, cmd = m.volumes.Update(msg)
	case Nodes:
		m.nodes, cmd = m.nodes.Update(msg)
	case Services:
		m.services, cmd = m.services.Update(msg)
	case Stacks:
		m.stacks, cmd = m.stacks.Update(msg)
	case Tasks, ServiceTasks, StackTasks:
		m.tasks, cmd = m.tasks.Update(msg)
	case ComposeProjects:
		m.composeProjects, cmd = m.composeProjects.Update(msg)
	case ComposeServices:
		m.composeServices, cmd = m.composeServices.Update(msg)
	}
	return m, cmd
}

func (m model) View() tea.View {
	var content string
	if !m.ready {
		content = m.renderLoadingScreen()
	} else if m.overlay == overlayLess {
		content = m.less.View()
	} else if m.overlay == overlayPrompt {
		content = m.renderMainScreenWithFooter(m.prompt.View())
	} else if m.overlay == overlayInputPrompt {
		content = m.inputPrompt.View()
	} else if m.overlay == overlayContainerMenu {
		content = m.containerMenu.View()
	} else if m.overlay == overlayCommandPalette {
		content = m.commandPalette.View()
	} else if m.overlay == overlayQuickPeek {
		content = m.quickPeek.View()
	} else {
		content = m.renderMainScreen()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.BackgroundColor = appui.DryTheme.Bg
	return v
}

// drainPendingComposeRefresh returns the compose reload that was deferred
// while a scan/drift cycle ran or a streaming viewer covered the view, and
// clears the flag. It returns nil when nothing was deferred, which is what
// keeps a completed cycle from chaining into another one forever. A pending
// refresh for a view the user has since left is dropped, not carried:
// switching back reloads that view anyway.
func (m *model) drainPendingComposeRefresh() tea.Cmd {
	if !m.composeRefreshPending || m.composeCycleInFlight || m.streamingViewerOpen() {
		return nil
	}
	m.composeRefreshPending = false
	switch m.view {
	case ComposeProjects:
		return loadComposeProjectsCmd(m.daemon)
	case ComposeServices:
		return loadComposeServicesCmd(m.daemon, m.selectedProject)
	}
	return nil
}

// streamingViewerOpen reports whether a live stream — container logs, or a
// compose up/down — is currently filling the less overlay. Background work
// whose only purpose is repainting the view underneath has nothing to show
// while it is true.
func (m model) streamingViewerOpen() bool {
	return m.overlay == overlayLess && m.streamReader != nil
}

func (m *model) closeActivityReader() {
	if m.activityReader != nil {
		_ = m.activityReader.Close()
		m.activityReader = nil
	}
}

func (m model) findContainerByID(id string) (*docker.Container, bool) {
	if c := m.daemon.ContainerByID(id); c != nil {
		return c, true
	}
	for _, c := range m.daemon.Containers(nil, docker.NoSort) {
		if c.ID == id {
			return c, true
		}
	}
	return nil, false
}

func (m model) findComposeProjectByName(name string) (*docker.ComposeProject, bool) {
	for _, p := range m.daemon.ComposeProjects() {
		if p.Name == name {
			project := p
			return &project, true
		}
	}
	return nil, false
}

func (m model) findComposeService(project, service string) (*docker.ComposeService, bool) {
	for _, svc := range m.daemon.ComposeServices(project) {
		if svc.Name == service {
			serviceCopy := svc
			return &serviceCopy, true
		}
	}
	return nil, false
}

func workspaceMapKeys[V any](m map[string]V, limit int) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return workspaceJoinLimited(keys, limit)
}

func (m model) switchView(target viewMode) (tea.Model, tea.Cmd) {
	if m.view == target {
		return m, nil
	}
	// Deactivate previous view
	if m.view == Monitor {
		m.monitor.StopAll()
	}
	m.previousView = m.view
	m.view = target
	if m.workspaceEnabled() {
		m.resetWorkspaceActivity()
	}
	// Monitor.Start() mutates the model (stores cancel funcs), so it must
	// run on the copy that gets returned — not inside loadViewData which
	// operates on a nested copy that gets discarded.
	if target == Monitor {
		cmds := m.monitor.Start()
		return m, tea.Batch(cmds...)
	}
	return m, m.loadViewData(target)
}

func (m model) loadViewData(v viewMode) tea.Cmd {
	if m.daemon == nil {
		return nil
	}
	switch v {
	case Main:
		return loadContainersCmd(m.daemon, m.containers.ShowAll(), m.containers.SortMode())
	case Images:
		return loadImagesCmd(m.daemon)
	case Networks:
		return loadNetworksCmd(m.daemon)
	case Volumes:
		return loadVolumesCmd(m.daemon)
	case DiskUsage:
		return loadDiskUsageCmd(m.daemon)
	case Monitor:
		// Monitor is handled directly in switchView to avoid the
		// value-receiver copy problem (Start mutates the model).
		return nil
	case Nodes:
		if m.swarmMode {
			return loadNodesCmd(m.daemon)
		}
	case Services:
		if m.swarmMode {
			return loadServicesCmd(m.daemon)
		}
	case Stacks:
		if m.swarmMode {
			return loadStacksCmd(m.daemon)
		}
	case ComposeProjects:
		return loadComposeProjectsCmd(m.daemon)
	case ComposeServices:
		return loadComposeServicesCmd(m.daemon, m.selectedProject)
	}
	return nil
}
