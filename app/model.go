package app

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/docker/docker/api/types/events"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-units"
	"github.com/moncho/dry/appui"
	appcompose "github.com/moncho/dry/appui/compose"
	appswarm "github.com/moncho/dry/appui/swarm"
	appworkspace "github.com/moncho/dry/appui/workspace"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui"
	"github.com/moncho/dry/version"
)

// Compile-time assertion: model implements tea.Model.
var _ tea.Model = model{}

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

type workspacePane int

const (
	workspacePaneNavigator workspacePane = iota
	workspacePaneContext
	workspacePaneActivity
)

type workspaceContextKind int

const (
	workspaceContextNone workspaceContextKind = iota
	workspaceContextContainer
	workspaceContextComposeProject
	workspaceContextComposeService
	workspaceContextMonitor
)

type workspaceContext struct {
	kind              workspaceContextKind
	title             string
	subtitle          string
	lines             []string
	containerID       string
	imageID           string
	monitorCID        string
	networkID         string
	nodeID            string
	project           string
	service           string
	serviceID         string
	stackName         string
	taskID            string
	volumeName        string
	monitorCPU        float64
	monitorMem        float64
	monitorMax        float64
	monitorPct        float64
	monitorCPUHistory []appui.MonitorPoint
	monitorMemHistory []appui.MonitorPoint
}

type monitorContainerLookup interface {
	ContainerByID(id string) *docker.Container
}

type model struct {
	// State
	view         viewMode
	previousView viewMode
	width        int
	height       int
	showHeader   bool
	ready        bool

	// Docker
	daemon       docker.ContainerDaemon
	config       Config
	swarmMode    bool
	eventsChan   <-chan events.Message
	eventsCancel context.CancelFunc

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

	// Loading animation
	loadingFrame int
	loadingFwd   bool

	// Splash screen
	splashDone bool
}

// NewModel creates a new top-level model.
func NewModel(cfg Config) model {
	return model{
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
		m.containers.SetDaemon(m.daemon)
		m.images.SetDaemon(m.daemon)
		m.networks.SetDaemon(m.daemon)
		m.volumes.SetDaemon(m.daemon)
		m.diskUsage.SetDaemon(m.daemon)
		m.monitor.SetDaemon(m.daemon)
		m.nodes.SetDaemon(m.daemon)
		m.services.SetDaemon(m.daemon)
		m.stacks.SetDaemon(m.daemon)
		m.tasks.SetDaemon(m.daemon)
		m.composeProjects.SetDaemon(m.daemon)
		m.resizeContentModels()
		m.header = appui.NewHeaderModel(m.daemon, m.width)
		eventsCtx, eventsCancel := context.WithCancel(context.Background())
		eventsCh, err := m.daemon.Events(eventsCtx)
		if err != nil {
			eventsCancel()
			m.messageBar.SetMessage(fmt.Sprintf("Docker events error: %s", err), 5*time.Second)
			return m, loadContainersCmd(m.daemon, m.containers.ShowAll(), m.containers.SortMode())
		}
		m.eventsChan = eventsCh
		m.eventsCancel = eventsCancel
		if m.config.MonitorMode {
			m2, cmd := m.switchView(Monitor)
			return m2, tea.Batch(cmd, listenDockerEvents(m.eventsChan))
		}
		return m, tea.Batch(
			loadContainersCmd(m.daemon, m.containers.ShowAll(), m.containers.SortMode()),
			listenDockerEvents(m.eventsChan),
		)

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
		cmd := m.monitor.UpdateStats(msg.CID, msg.Stats, msg.StatsCh)
		m.refreshPinnedWorkspaceContext()
		return m, tea.Batch(cmd, m.workspaceMonitorActivityCmd(msg.CID))

	case appui.MonitorErrorMsg:
		m.monitor.RemoveContainer(msg.CID)
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
		return m, nil

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
				if m.view == ComposeProjects {
					cmds = append(cmds, loadComposeProjectsCmd(m.daemon))
				}
				if m.view == ComposeServices {
					cmds = append(cmds, loadComposeServicesCmd(m.daemon, m.selectedProject))
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
		m.less = appui.NewLessModel()
		m.less.SetSize(m.width, m.height)
		m.less.SetContent(msg.content, msg.title)
		m.less.SetFollowing(true)
		m.overlay = overlayLess
		m.streamReader = msg.reader
		return m, readLogStreamCmd(msg.reader)

	case appendLessMsg:
		if m.overlay == overlayLess && m.streamReader != nil {
			m.less.AppendContent(msg.content)
			return m, readLogStreamCmd(msg.reader)
		}
		// Overlay was closed, clean up the reader
		if msg.reader != nil {
			_ = msg.reader.Close()
		}
		return m, nil

	case streamClosedMsg:
		m.streamReader = nil
		return m, nil

	case appui.CloseOverlayMsg:
		m.overlay = overlayNone
		m.eventsLive = false
		if m.streamReader != nil {
			_ = m.streamReader.Close()
			m.streamReader = nil
		}
		return m, nil

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

func (m model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Quit keys always handled regardless of filter state
	switch msg.String() {
	case "ctrl+c", "Q":
		m.monitor.StopAll()
		if m.streamReader != nil {
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
		return m.switchView(Nodes)
	case "6":
		return m.switchView(Services)
	case "7":
		return m.switchView(Stacks)
	case "8":
		return m.switchView(ComposeProjects)
	case "esc":
		if m.workspaceEnabled() && m.pinnedContext != nil {
			return m.clearPinnedContext(), nil
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

	case Images:
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

	case Networks:
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

	case Volumes:
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

	case DiskUsage:
		switch msg.String() {
		case "p", "P":
			return m.showPrompt("Prune all unused Docker resources?", "prune", ""), nil
		case "f5":
			return m, loadDiskUsageCmd(m.daemon)
		}
		var cmd tea.Cmd
		m.diskUsage, cmd = m.diskUsage.Update(msg)
		return m, cmd

	case Monitor:
		var cmd tea.Cmd
		m.monitor, cmd = m.monitor.Update(msg)
		return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

	case Nodes:
		if m.swarmMode {
			switch msg.String() {
			case "enter":
				if n := m.nodes.SelectedNode(); n != nil {
					m.previousView = m.view
					m.view = Tasks
					return m, loadNodeTasksCmd(m.daemon, n.ID)
				}
				return m, nil
			case "i", "I":
				if n := m.nodes.SelectedNode(); n != nil {
					return m, inspectNodeCmd(m.daemon, n.ID)
				}
				return m, nil
			case "ctrl+a":
				if n := m.nodes.SelectedNode(); n != nil {
					return m, m.cycleNodeAvailability(n.ID)
				}
				return m, nil
			case "f5":
				return m, loadNodesCmd(m.daemon)
			}
		}
		var cmd tea.Cmd
		m.nodes, cmd = m.nodes.Update(msg)
		return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

	case Services:
		if m.swarmMode {
			switch msg.String() {
			case "enter":
				if s := m.services.SelectedService(); s != nil {
					m.previousView = m.view
					m.view = ServiceTasks
					return m, loadServiceTasksCmd(m.daemon, s.ID)
				}
				return m, nil
			case "i", "I":
				if s := m.services.SelectedService(); s != nil {
					return m, inspectServiceCmd(m.daemon, s.ID)
				}
				return m, nil
			case "l", "L":
				if s := m.services.SelectedService(); s != nil {
					return m, showServiceLogsCmd(m.daemon, s.ID)
				}
				return m, nil
			case "ctrl+r":
				if s := m.services.SelectedService(); s != nil {
					return m.showPrompt(
						fmt.Sprintf("Remove service %s?", s.Spec.Name),
						"service-rm", s.ID,
					), nil
				}
				return m, nil
			case "ctrl+s":
				if s := m.services.SelectedService(); s != nil {
					var cmd tea.Cmd
					m.inputPrompt, cmd = appui.NewInputPromptModel(
						fmt.Sprintf("Scale service %s to replicas:", s.Spec.Name),
						"number", "service-scale", s.ID,
					)
					m.inputPrompt.SetSize(m.width, m.height)
					m.overlay = overlayInputPrompt
					return m, cmd
				}
				return m, nil
			case "ctrl+u":
				if s := m.services.SelectedService(); s != nil {
					return m.showPrompt(
						fmt.Sprintf("Force update service %s?", s.Spec.Name),
						"service-update", s.ID,
					), nil
				}
				return m, nil
			case "f5":
				return m, loadServicesCmd(m.daemon)
			}
		}
		var cmd tea.Cmd
		m.services, cmd = m.services.Update(msg)
		return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

	case Stacks:
		if m.swarmMode {
			switch msg.String() {
			case "enter":
				if s := m.stacks.SelectedStack(); s != nil {
					m.previousView = m.view
					m.view = StackTasks
					return m, loadStackTasksCmd(m.daemon, s.Name)
				}
				return m, nil
			case "ctrl+r":
				if s := m.stacks.SelectedStack(); s != nil {
					return m.showPrompt(
						fmt.Sprintf("Remove stack %s?", s.Name),
						"stack-rm", s.Name,
					), nil
				}
				return m, nil
			case "f5":
				return m, loadStacksCmd(m.daemon)
			}
		}
		var cmd tea.Cmd
		m.stacks, cmd = m.stacks.Update(msg)
		return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())

	case ComposeProjects:
		switch msg.String() {
		case "enter":
			// Service row: inspect the service's first container
			if svc := m.composeProjects.SelectedService(); svc != nil {
				return m, inspectComposeServiceCmd(m.daemon, svc.Project, svc.Name)
			}
			// Project row: drill into project resources
			if p := m.composeProjects.SelectedProject(); p != nil {
				m.previousView = m.view
				m.view = ComposeServices
				m.selectedProject = p.Name
				return m, loadComposeServicesCmd(m.daemon, p.Name)
			}
			return m, nil
		case "l", "L":
			if svc := m.composeProjects.SelectedService(); svc != nil {
				return m, showComposeLogsCmd(m.daemon, svc.Project, svc.Name)
			}
			if p := m.composeProjects.SelectedProject(); p != nil {
				return m, showComposeLogsCmd(m.daemon, p.Name, "")
			}
			return m, nil
		case "f5":
			return m, loadComposeProjectsCmd(m.daemon)
		case "ctrl+t":
			if p := m.composeProjects.SelectedProject(); p != nil {
				return m.showPrompt(fmt.Sprintf("Stop project %s?", p.Name),
					"compose-project-stop", p.Name), nil
			}
		case "ctrl+r":
			if p := m.composeProjects.SelectedProject(); p != nil {
				return m.showPrompt(fmt.Sprintf("Restart project %s?", p.Name),
					"compose-project-restart", p.Name), nil
			}
		case "ctrl+e":
			if p := m.composeProjects.SelectedProject(); p != nil {
				return m.showPrompt(fmt.Sprintf("Remove project %s containers?", p.Name),
					"compose-project-rm", p.Name), nil
			}
		}
		var cmd tea.Cmd
		m.composeProjects, cmd = m.composeProjects.Update(msg)
		return m, cmd

	case ComposeServices:
		switch msg.String() {
		case "esc":
			if m.workspaceEnabled() && m.pinnedContext != nil {
				return m.clearPinnedContext(), nil
			}
			m.view = ComposeProjects
			return m, loadComposeProjectsCmd(m.daemon)
		case "enter":
			if svc := m.composeServices.SelectedService(); svc != nil {
				return m, inspectComposeServiceCmd(m.daemon, svc.Project, svc.Name)
			}
			if n := m.composeServices.SelectedNetwork(); n != nil {
				return m, inspectNetworkCmd(m.daemon, n.Name)
			}
			if v := m.composeServices.SelectedVolume(); v != nil {
				return m, inspectVolumeCmd(m.daemon, v.Name)
			}
			return m, nil
		case "l", "L":
			if svc := m.composeServices.SelectedService(); svc != nil {
				return m, showComposeLogsCmd(m.daemon, svc.Project, svc.Name)
			}
			return m, nil
		case "f5":
			return m, loadComposeServicesCmd(m.daemon, m.selectedProject)
		case "ctrl+s":
			if svc := m.composeServices.SelectedService(); svc != nil {
				return m.showPrompt(fmt.Sprintf("Start service %s?", svc.Name),
					"compose-start", svc.Project+"/"+svc.Name), nil
			}
		case "ctrl+t":
			if svc := m.composeServices.SelectedService(); svc != nil {
				return m.showPrompt(fmt.Sprintf("Stop service %s?", svc.Name),
					"compose-stop", svc.Project+"/"+svc.Name), nil
			}
		case "ctrl+r":
			if svc := m.composeServices.SelectedService(); svc != nil {
				return m.showPrompt(fmt.Sprintf("Restart service %s?", svc.Name),
					"compose-restart", svc.Project+"/"+svc.Name), nil
			}
		case "ctrl+e":
			if svc := m.composeServices.SelectedService(); svc != nil {
				return m.showPrompt(fmt.Sprintf("Remove service %s containers?", svc.Name),
					"compose-rm", svc.Project+"/"+svc.Name), nil
			}
		}
		var cmd tea.Cmd
		m.composeServices, cmd = m.composeServices.Update(msg)
		return m, cmd

	case ServiceTasks, Tasks, StackTasks:
		switch msg.String() {
		case "esc":
			m.view = m.previousView
			return m, m.loadViewData(m.view)
		}
		var cmd tea.Cmd
		m.tasks, cmd = m.tasks.Update(msg)
		return m, tea.Batch(cmd, m.workspaceSelectionActivityCmd())
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

	renderBindings := func(bindings []key.Binding) string {
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
			if !first {
				b.WriteString(sepStyle.Render("  \u00b7  "))
			}
			first = false
			b.WriteString(keyStyle.Render(kb.Help().Key))
			b.WriteString(footerBg.Render(" "))
			b.WriteString(descStyle.Render(kb.Help().Desc))
		}
		return b.String()
	}

	line := renderBindings(bindings)
	w := ansi.StringWidth(line)
	if w > m.width {
		line = ansi.Truncate(line, m.width, "")
	} else if w < m.width {
		line += footerBg.Render(strings.Repeat(" ", m.width-w))
	}
	return line
}

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

func (m model) workspaceEnabled() bool {
	return m.config.WorkspaceMode
}

func (m model) workspaceCompactMode() bool {
	return m.width < 100 || m.contentHeight() < 12
}

func (m model) workspaceLayout() (navigatorW, contextW, topH, activityH int) {
	contentH := m.contentHeight()
	usableH := contentH - 1 // reserve one line for workspace tabs
	if usableH < 1 {
		return m.width, 0, 0, 0
	}
	if m.workspaceCompactMode() {
		return m.width, 0, usableH, 0
	}
	if usableH < 7 {
		return m.width, 0, usableH, 0
	}

	activityH = 8
	if usableH < 18 {
		activityH = 5
	}
	topH = usableH - activityH
	if topH < 3 {
		topH = usableH
		activityH = 0
	}

	// In workspace mode, favor the activity pane in the container view.
	// ContainersModel currently needs 9 lines to show at most 5 data rows:
	// 5 rows + widget/table framing.
	if m.view == Main && topH > 9 {
		topH = 9
		activityH = usableH - topH
	}
	if m.view == Monitor && topH > 10 {
		topH = 10
		activityH = usableH - topH
	}

	navigatorW = m.width * 58 / 100
	if navigatorW < 40 {
		navigatorW = 40
	}
	if navigatorW > m.width-24 {
		navigatorW = m.width - 24
	}
	if navigatorW < 1 {
		navigatorW = m.width
	}
	contextW = m.width - navigatorW
	if contextW < 1 {
		contextW = 1
	}
	return navigatorW, contextW, topH, activityH
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

func (m model) renderWorkspaceBody() string {
	navigatorW, _, topH, activityH := m.workspaceLayout()
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
	if navigatorW <= 0 {
		return lipgloss.JoinVertical(lipgloss.Left, tabs, top, activity)
	}
	return lipgloss.JoinVertical(lipgloss.Left, tabs, top, activity)
}

func (m *model) populateWorkspaceContextPane() {
	context := m.pinnedContext
	if context == nil {
		if current, ok := m.currentWorkspacePreview(); ok {
			context = &current
		}
	}
	if context != nil {
		m.workspaceContext.SetEmptyMessage("")
		if m.pinnedContext != nil {
			m.workspaceContext.SetMode("pinned")
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
	if m.pinnedContext != nil {
		cleared := m.clearPinnedContext()
		return cleared, cleared.workspaceSelectionActivityCmd()
	}
	ctx, ok := m.currentWorkspacePreview()
	if !ok {
		return m, nil
	}
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

func (m *model) closeActivityReader() {
	if m.activityReader != nil {
		_ = m.activityReader.Close()
		m.activityReader = nil
	}
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

func workspaceContextFromContainer(c *docker.Container) workspaceContext {
	name := shortID(c.ID)
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}
	lines := []string{
		fmt.Sprintf("id: %s", shortID(c.ID)),
		fmt.Sprintf("status: %s", c.Status),
		fmt.Sprintf("image: %s", c.Image),
	}
	if c.Created > 0 {
		lines = append(lines, fmt.Sprintf("created: %s", workspaceFormatUnix(c.Created)))
	}
	if c.Detail.ContainerJSONBase != nil && c.Detail.State != nil {
		if status := c.Detail.State.Status; status != "" {
			lines = append(lines, fmt.Sprintf("state: %s", status))
		}
		if c.Detail.State.Health != nil && c.Detail.State.Health.Status != "" {
			lines = append(lines, fmt.Sprintf("health: %s", c.Detail.State.Health.Status))
		}
		if started := workspaceFormatTimestamp(c.Detail.State.StartedAt); started != "" {
			lines = append(lines, fmt.Sprintf("started: %s", started))
		}
		if finished := workspaceFormatTimestamp(c.Detail.State.FinishedAt); finished != "" {
			lines = append(lines, fmt.Sprintf("finished: %s", finished))
		}
	}
	if c.Detail.ContainerJSONBase != nil && c.Detail.RestartCount > 0 {
		lines = append(lines, fmt.Sprintf("restarts: %d", c.Detail.RestartCount))
	}
	if project := c.Labels["com.docker.compose.project"]; project != "" {
		lines = append(lines, fmt.Sprintf("compose project: %s", project))
	}
	if service := c.Labels["com.docker.compose.service"]; service != "" {
		lines = append(lines, fmt.Sprintf("compose service: %s", service))
	}
	if c.Command != "" {
		lines = append(lines, fmt.Sprintf("command: %s", c.Command))
	}
	if len(c.Ports) > 0 {
		lines = append(lines, fmt.Sprintf("ports: %s", workspaceContainerPorts(c)))
	}
	if c.Detail.Config != nil {
		if c.Detail.Config.User != "" {
			lines = append(lines, fmt.Sprintf("user: %s", c.Detail.Config.User))
		}
		if c.Detail.Config.WorkingDir != "" {
			lines = append(lines, fmt.Sprintf("workdir: %s", c.Detail.Config.WorkingDir))
		}
		if envs := len(c.Detail.Config.Env); envs > 0 {
			lines = append(lines, fmt.Sprintf("env: %d vars", envs))
		}
	}
	if mounts := len(c.Detail.Mounts); mounts > 0 {
		lines = append(lines, fmt.Sprintf("mounts: %d", mounts))
		if targets := workspaceContainerMountTargets(c); targets != "" {
			lines = append(lines, fmt.Sprintf("mount targets: %s", targets))
		}
	}
	if networks := workspaceContainerNetworkCount(c); networks > 0 {
		lines = append(lines, fmt.Sprintf("networks: %d", networks))
		if names := workspaceContainerNetworkNames(c); names != "" {
			lines = append(lines, fmt.Sprintf("network names: %s", names))
		}
	}
	if labels := len(c.Labels); labels > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", labels))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(c.Labels, 6)))
	}
	return workspaceContext{
		kind:        workspaceContextContainer,
		title:       name,
		subtitle:    "Container",
		lines:       lines,
		containerID: c.ID,
	}
}

func workspaceContextFromComposeProject(p docker.ComposeProject) workspaceContext {
	lines := []string{
		fmt.Sprintf("services: %d", p.Services),
		fmt.Sprintf("containers: %d", p.Containers),
		fmt.Sprintf("running: %d", p.Running),
		fmt.Sprintf("exited: %d", p.Exited),
		fmt.Sprintf("health: %s", workspaceRunningHealth(p.Running, p.Containers)),
	}
	if p.Containers > 0 {
		lines = append(lines, fmt.Sprintf("status ratio: %d/%d running", p.Running, p.Containers))
	}
	return workspaceContext{
		kind:     workspaceContextComposeProject,
		title:    p.Name,
		subtitle: "Compose Project",
		lines:    lines,
		project: p.Name,
	}
}

func workspaceContextFromComposeService(s docker.ComposeService) workspaceContext {
	lines := []string{
		fmt.Sprintf("project: %s", s.Project),
		fmt.Sprintf("service: %s", s.Name),
		fmt.Sprintf("containers: %d", s.Containers),
		fmt.Sprintf("running: %d", s.Running),
		fmt.Sprintf("exited: %d", s.Exited),
	}
	if s.Containers > 0 {
		lines = append(lines, fmt.Sprintf("status ratio: %d/%d running", s.Running, s.Containers))
	}
	if s.Image != "" {
		lines = append(lines, fmt.Sprintf("image: %s", s.Image))
	}
	if s.Health != "" {
		lines = append(lines, fmt.Sprintf("health: %s", s.Health))
	}
	if s.Ports != "" {
		lines = append(lines, fmt.Sprintf("ports: %s", s.Ports))
	}
	lines = append(lines, fmt.Sprintf("health summary: %s", workspaceRunningHealth(s.Running, s.Containers)))
	return workspaceContext{
		kind:     workspaceContextComposeService,
		title:    s.Name,
		subtitle: "Compose Service",
		lines:    lines,
		project:  s.Project,
		service:  s.Name,
	}
}

func workspaceContextFromImage(img image.Summary) workspaceContext {
	title := docker.TruncateID(docker.ImageID(img.ID))
	subtitle := "Image"
	if len(img.RepoTags) > 0 && img.RepoTags[0] != "<none>:<none>" {
		title = img.RepoTags[0]
	}
	lines := []string{
		fmt.Sprintf("id: %s", docker.TruncateID(docker.ImageID(img.ID))),
	}
	if img.ParentID != "" {
		lines = append(lines, fmt.Sprintf("parent: %s", docker.TruncateID(docker.ImageID(img.ParentID))))
	}
	if len(img.RepoTags) > 0 {
		lines = append(lines, fmt.Sprintf("tag count: %d", len(img.RepoTags)))
		lines = append(lines, fmt.Sprintf("tags: %s", strings.Join(img.RepoTags, ", ")))
	}
	if len(img.RepoDigests) > 0 {
		lines = append(lines, fmt.Sprintf("digests: %d", len(img.RepoDigests)))
		lines = append(lines, fmt.Sprintf("digest refs: %s", workspaceJoinLimited(img.RepoDigests, 3)))
	}
	if len(img.Manifests) > 0 {
		lines = append(lines, fmt.Sprintf("manifests: %d", len(img.Manifests)))
	}
	if len(img.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", len(img.Labels)))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(img.Labels, 6)))
	}
	lines = append(lines,
		fmt.Sprintf("created: %s", workspaceFormatUnix(img.Created)),
		fmt.Sprintf("size: %s", units.BytesSize(float64(img.Size))),
	)
	if img.SharedSize > 0 {
		lines = append(lines, fmt.Sprintf("shared size: %s", units.BytesSize(float64(img.SharedSize))))
	}
	if img.Containers > 0 {
		lines = append(lines, fmt.Sprintf("used by: %d containers", img.Containers))
	}
	return workspaceContext{
		title:    title,
		subtitle: subtitle,
		lines:    lines,
		imageID:  img.ID,
	}
}

func workspaceContextFromNetwork(n network.Inspect) workspaceContext {
	title := n.Name
	if title == "" {
		title = docker.TruncateID(n.ID)
	}
	lines := []string{
		fmt.Sprintf("id: %s", docker.TruncateID(n.ID)),
		fmt.Sprintf("driver: %s", n.Driver),
		fmt.Sprintf("scope: %s", n.Scope),
		fmt.Sprintf("containers: %d", len(n.Containers)),
	}
	if !n.Created.IsZero() {
		lines = append(lines, fmt.Sprintf("created: %s", n.Created.Format("2006-01-02 15:04")))
	}
	if n.IPAM.Driver != "" {
		lines = append(lines, fmt.Sprintf("ipam: %s", n.IPAM.Driver))
	}
	if len(n.IPAM.Config) > 0 && n.IPAM.Config[0].Subnet != "" {
		lines = append(lines, fmt.Sprintf("subnet: %s", n.IPAM.Config[0].Subnet))
	}
	if len(n.IPAM.Config) > 0 && n.IPAM.Config[0].Gateway != "" {
		lines = append(lines, fmt.Sprintf("gateway: %s", n.IPAM.Config[0].Gateway))
	}
	if n.Internal {
		lines = append(lines, "internal: true")
	}
	if n.Attachable {
		lines = append(lines, "attachable: true")
	}
	if n.Ingress {
		lines = append(lines, "ingress: true")
	}
	if n.ConfigOnly {
		lines = append(lines, "config only: true")
	}
	if !n.EnableIPv4 {
		lines = append(lines, "ipv4: disabled")
	}
	if n.EnableIPv6 {
		lines = append(lines, "ipv6: enabled")
	}
	if services := len(n.Services); services > 0 {
		lines = append(lines, fmt.Sprintf("services: %d", services))
	}
	if options := len(n.Options); options > 0 {
		lines = append(lines, fmt.Sprintf("options: %d", options))
		lines = append(lines, fmt.Sprintf("option keys: %s", workspaceMapKeys(n.Options, 6)))
	}
	if labels := len(n.Labels); labels > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", labels))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(n.Labels, 6)))
	}
	if attached := workspaceAttachedNetworkNames(n); attached != "" {
		lines = append(lines, fmt.Sprintf("attached: %s", attached))
	}
	return workspaceContext{
		title:     title,
		subtitle:  "Network",
		lines:     lines,
		networkID: n.ID,
	}
}

func workspaceContextFromVolume(v *volume.Volume) workspaceContext {
	if v == nil {
		return workspaceContext{}
	}
	lines := []string{
		fmt.Sprintf("name: %s", v.Name),
		fmt.Sprintf("driver: %s", v.Driver),
		fmt.Sprintf("mountpoint: %s", v.Mountpoint),
	}
	if v.CreatedAt != "" {
		lines = append(lines, fmt.Sprintf("created: %s", v.CreatedAt))
	}
	if v.Scope != "" {
		lines = append(lines, fmt.Sprintf("scope: %s", v.Scope))
	}
	if len(v.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", len(v.Labels)))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(v.Labels, 6)))
	}
	if len(v.Options) > 0 {
		lines = append(lines, fmt.Sprintf("options: %d", len(v.Options)))
		lines = append(lines, fmt.Sprintf("option keys: %s", workspaceMapKeys(v.Options, 6)))
	}
	return workspaceContext{
		title:      v.Name,
		subtitle:   "Volume",
		lines:      lines,
		volumeName: v.Name,
	}
}

func workspaceContextFromStats(s *docker.Stats, lookup monitorContainerLookup, series appui.MonitorSeries) workspaceContext {
	if s == nil {
		return workspaceContext{}
	}
	title := s.CID
	lines := []string{
		fmt.Sprintf("container: %s", s.CID),
		fmt.Sprintf("command: %s", s.Command),
		fmt.Sprintf("net io: %s / %s", units.BytesSize(s.NetworkRx), units.BytesSize(s.NetworkTx)),
		fmt.Sprintf("block io: %s / %s", units.BytesSize(s.BlockRead), units.BytesSize(s.BlockWrite)),
		fmt.Sprintf("pids: %d", s.PidsCurrent),
	}
	if c := workspaceMonitorContainer(lookup, s.CID); c != nil {
		if name := workspaceContainerPrimaryName(c); name != "" {
			title = name
			lines = append([]string{fmt.Sprintf("name: %s", name)}, lines...)
		}
		if c.Status != "" {
			lines = append(lines, fmt.Sprintf("status: %s", c.Status))
		}
		if c.Image != "" {
			lines = append(lines, fmt.Sprintf("image: %s", c.Image))
		}
		if c.Detail.ContainerJSONBase != nil && c.Detail.State != nil &&
			c.Detail.State.Health != nil && c.Detail.State.Health.Status != "" {
			lines = append(lines, fmt.Sprintf("health: %s", c.Detail.State.Health.Status))
		}
		if c.Detail.ContainerJSONBase != nil && c.Detail.RestartCount > 0 {
			lines = append(lines, fmt.Sprintf("restarts: %d", c.Detail.RestartCount))
		}
		if ports := workspaceContainerPorts(c); ports != "" {
			lines = append(lines, fmt.Sprintf("ports: %s", ports))
		}
		if c.Detail.Config != nil {
			if c.Detail.Config.User != "" {
				lines = append(lines, fmt.Sprintf("user: %s", c.Detail.Config.User))
			}
			if c.Detail.Config.WorkingDir != "" {
				lines = append(lines, fmt.Sprintf("workdir: %s", c.Detail.Config.WorkingDir))
			}
		}
		if project := c.Labels["com.docker.compose.project"]; project != "" {
			lines = append(lines, fmt.Sprintf("compose project: %s", project))
		}
		if service := c.Labels["com.docker.compose.service"]; service != "" {
			lines = append(lines, fmt.Sprintf("compose service: %s", service))
		}
		if networks := workspaceContainerNetworkNames(c); networks != "" {
			lines = append(lines, fmt.Sprintf("network names: %s", networks))
		}
		if mounts := workspaceContainerMountTargets(c); mounts != "" {
			lines = append(lines, fmt.Sprintf("mount targets: %s", mounts))
		}
		if labels := len(c.Labels); labels > 0 {
			lines = append(lines, fmt.Sprintf("labels: %d", labels))
		}
	}
	lines = append(lines, workspaceDockerStatsLines(s.Stats)...)
	return workspaceContext{
		kind:              workspaceContextMonitor,
		title:             title,
		subtitle:          "Monitor",
		lines:             lines,
		monitorCID:        s.CID,
		monitorCPU:        s.CPUPercentage,
		monitorMem:        s.Memory,
		monitorMax:        s.MemoryLimit,
		monitorPct:        s.MemoryPercentage,
		monitorCPUHistory: append([]appui.MonitorPoint(nil), series.CPU...),
		monitorMemHistory: append([]appui.MonitorPoint(nil), series.Memory...),
	}
}

func workspaceMonitorContainer(lookup monitorContainerLookup, id string) *docker.Container {
	if lookup == nil {
		return nil
	}
	return lookup.ContainerByID(id)
}

func workspaceContainerPrimaryName(c *docker.Container) string {
	if c == nil || len(c.Names) == 0 {
		return ""
	}
	return strings.TrimPrefix(c.Names[0], "/")
}

func workspaceContainerPorts(c *docker.Container) string {
	if c == nil || len(c.Ports) == 0 {
		return ""
	}
	var ports []string
	for _, p := range c.Ports {
		if p.PublicPort != 0 {
			ports = append(ports, fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type))
			continue
		}
		ports = append(ports, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
	}
	return strings.Join(ports, ", ")
}

func workspaceContextFromNode(n swarm.Node) workspaceContext {
	lines := []string{
		fmt.Sprintf("id: %s", docker.TruncateID(n.ID)),
		fmt.Sprintf("hostname: %s", n.Description.Hostname),
		fmt.Sprintf("role: %s", n.Spec.Role),
		fmt.Sprintf("availability: %s", n.Spec.Availability),
		fmt.Sprintf("status: %s", n.Status.State),
		fmt.Sprintf("cpu: %d", n.Description.Resources.NanoCPUs/1e9),
		fmt.Sprintf("memory: %s", units.BytesSize(float64(n.Description.Resources.MemoryBytes))),
	}
	if n.Description.Engine.EngineVersion != "" {
		lines = append(lines, fmt.Sprintf("engine: %s", n.Description.Engine.EngineVersion))
	}
	if n.Description.Platform.OS != "" || n.Description.Platform.Architecture != "" {
		lines = append(lines, fmt.Sprintf("platform: %s/%s", n.Description.Platform.OS, n.Description.Platform.Architecture))
	}
	if n.ManagerStatus != nil {
		lines = append(lines, fmt.Sprintf("manager: %s", n.ManagerStatus.Reachability))
		if n.ManagerStatus.Leader {
			lines = append(lines, "leader: true")
		}
		if n.ManagerStatus.Addr != "" {
			lines = append(lines, fmt.Sprintf("manager addr: %s", n.ManagerStatus.Addr))
		}
	}
	if len(n.Spec.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", len(n.Spec.Labels)))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(n.Spec.Labels, 6)))
	}
	return workspaceContext{
		title:    n.Description.Hostname,
		subtitle: "Node",
		lines:    lines,
		nodeID:   n.ID,
	}
}

func workspaceContextFromSwarmService(s swarm.Service) workspaceContext {
	mode := "global"
	replicas := "global"
	if s.Spec.Mode.Replicated != nil {
		mode = "replicated"
		if s.Spec.Mode.Replicated.Replicas != nil {
			replicas = fmt.Sprintf("%d", *s.Spec.Mode.Replicated.Replicas)
		} else {
			replicas = "0"
		}
	}
	imageRef := ""
	if s.Spec.TaskTemplate.ContainerSpec != nil {
		imageRef = s.Spec.TaskTemplate.ContainerSpec.Image
	}
	lines := []string{
		fmt.Sprintf("id: %s", docker.TruncateID(s.ID)),
		fmt.Sprintf("mode: %s", mode),
		fmt.Sprintf("replicas: %s", replicas),
	}
	if s.ServiceStatus != nil {
		lines = append(lines, fmt.Sprintf("tasks: %d/%d running", s.ServiceStatus.RunningTasks, s.ServiceStatus.DesiredTasks))
	}
	if imageRef != "" {
		lines = append(lines, fmt.Sprintf("image: %s", imageRef))
	}
	if spec := s.Spec.TaskTemplate.ContainerSpec; spec != nil {
		if len(spec.Command) > 0 {
			lines = append(lines, fmt.Sprintf("command: %s", workspaceJoinLimited(spec.Command, 4)))
		}
		if spec.User != "" {
			lines = append(lines, fmt.Sprintf("user: %s", spec.User))
		}
		if spec.Dir != "" {
			lines = append(lines, fmt.Sprintf("workdir: %s", spec.Dir))
		}
		if len(spec.Env) > 0 {
			lines = append(lines, fmt.Sprintf("env: %d vars", len(spec.Env)))
		}
		if len(spec.Mounts) > 0 {
			lines = append(lines, fmt.Sprintf("mounts: %d", len(spec.Mounts)))
		}
		if len(spec.Secrets) > 0 {
			lines = append(lines, fmt.Sprintf("secrets: %d", len(spec.Secrets)))
		}
		if len(spec.Configs) > 0 {
			lines = append(lines, fmt.Sprintf("configs: %d", len(spec.Configs)))
		}
		if len(spec.Labels) > 0 {
			lines = append(lines, fmt.Sprintf("container labels: %d", len(spec.Labels)))
		}
	}
	if len(s.Endpoint.Ports) > 0 {
		lines = append(lines, fmt.Sprintf("ports: %s", workspaceFormatSwarmPorts(s.Endpoint.Ports)))
	}
	if len(s.Spec.TaskTemplate.Networks) > 0 {
		lines = append(lines, fmt.Sprintf("networks: %d", len(s.Spec.TaskTemplate.Networks)))
	}
	if s.Spec.TaskTemplate.Placement != nil && len(s.Spec.TaskTemplate.Placement.Constraints) > 0 {
		lines = append(lines, fmt.Sprintf("constraints: %s", workspaceJoinLimited(s.Spec.TaskTemplate.Placement.Constraints, 4)))
	}
	if s.UpdateStatus != nil {
		lines = append(lines, fmt.Sprintf("update: %s", s.UpdateStatus.State))
		if s.UpdateStatus.Message != "" {
			lines = append(lines, fmt.Sprintf("update message: %s", s.UpdateStatus.Message))
		}
	}
	if s.Spec.UpdateConfig != nil {
		lines = append(lines, fmt.Sprintf("update policy: %s", s.Spec.UpdateConfig.FailureAction))
	}
	if s.Spec.RollbackConfig != nil {
		lines = append(lines, fmt.Sprintf("rollback policy: %s", s.Spec.RollbackConfig.FailureAction))
	}
	if len(s.Spec.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("labels: %d", len(s.Spec.Labels)))
		lines = append(lines, fmt.Sprintf("label keys: %s", workspaceMapKeys(s.Spec.Labels, 6)))
	}
	return workspaceContext{
		title:     s.Spec.Name,
		subtitle:  "Swarm Service",
		lines:     lines,
		serviceID: s.ID,
	}
}

func workspaceContextFromStack(s docker.Stack) workspaceContext {
	lines := []string{
		fmt.Sprintf("orchestrator: %s", s.Orchestrator),
		fmt.Sprintf("services: %d", s.Services),
		fmt.Sprintf("networks: %d", s.Networks),
		fmt.Sprintf("configs: %d", s.Configs),
		fmt.Sprintf("secrets: %d", s.Secrets),
	}
	if s.Services > 0 {
		lines = append(lines, fmt.Sprintf("network ratio: %d services / %d networks", s.Services, workspaceMaxInt(s.Networks, 1)))
	}
	return workspaceContext{
		title:     s.Name,
		subtitle:  "Stack",
		stackName: s.Name,
		lines:     lines,
	}
}

func workspaceContextFromTask(t swarm.Task) workspaceContext {
	title := docker.TruncateID(t.ID)
	if t.Spec.ContainerSpec != nil && t.Spec.ContainerSpec.Hostname != "" {
		title = t.Spec.ContainerSpec.Hostname
	}
	lines := []string{
		fmt.Sprintf("id: %s", docker.TruncateID(t.ID)),
		fmt.Sprintf("desired: %s", t.DesiredState),
		fmt.Sprintf("current: %s", t.Status.State),
	}
	if !t.Status.Timestamp.IsZero() {
		lines = append(lines, fmt.Sprintf("updated: %s", t.Status.Timestamp.Format("2006-01-02 15:04")))
	}
	if t.Slot != 0 {
		lines = append(lines, fmt.Sprintf("slot: %d", t.Slot))
	}
	if t.ServiceID != "" {
		lines = append(lines, fmt.Sprintf("service: %s", docker.TruncateID(t.ServiceID)))
	}
	if t.NodeID != "" {
		lines = append(lines, fmt.Sprintf("node: %s", docker.TruncateID(t.NodeID)))
	}
	if t.Spec.ContainerSpec != nil {
		if t.Spec.ContainerSpec.Image != "" {
			lines = append(lines, fmt.Sprintf("image: %s", t.Spec.ContainerSpec.Image))
		}
		if len(t.Spec.ContainerSpec.Command) > 0 {
			lines = append(lines, fmt.Sprintf("command: %s", workspaceJoinLimited(t.Spec.ContainerSpec.Command, 4)))
		}
	}
	if t.Status.ContainerStatus != nil {
		if t.Status.ContainerStatus.ContainerID != "" {
			lines = append(lines, fmt.Sprintf("container: %s", docker.TruncateID(t.Status.ContainerStatus.ContainerID)))
		}
		if t.Status.ContainerStatus.ExitCode != 0 {
			lines = append(lines, fmt.Sprintf("exit code: %d", t.Status.ContainerStatus.ExitCode))
		}
	}
	if len(t.NetworksAttachments) > 0 {
		lines = append(lines, fmt.Sprintf("networks: %d", len(t.NetworksAttachments)))
	}
	if len(t.Status.PortStatus.Ports) > 0 {
		lines = append(lines, fmt.Sprintf("ports: %s", workspaceFormatSwarmPorts(t.Status.PortStatus.Ports)))
	}
	if t.Status.Message != "" {
		lines = append(lines, fmt.Sprintf("message: %s", t.Status.Message))
	}
	if t.Status.Err != "" {
		lines = append(lines, fmt.Sprintf("error: %s", t.Status.Err))
	}
	return workspaceContext{
		title:    title,
		subtitle: "Task",
		lines:    lines,
		taskID:   t.ID,
	}
}

func workspaceContainerNetworkCount(c *docker.Container) int {
	if c == nil || c.Detail.NetworkSettings == nil {
		return 0
	}
	return len(c.Detail.NetworkSettings.Networks)
}

func workspaceRunningHealth(running, total int) string {
	if total == 0 {
		return "empty"
	}
	if running == total {
		return "healthy"
	}
	if running == 0 {
		return "stopped"
	}
	return "degraded"
}

func workspaceAttachedNetworkNames(n network.Inspect) string {
	names := make([]string, 0, len(n.Containers))
	for _, endpoint := range n.Containers {
		if endpoint.Name != "" {
			names = append(names, endpoint.Name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	slices.Sort(names)
	if len(names) > 3 {
		return strings.Join(names[:3], ", ") + fmt.Sprintf(" +%d", len(names)-3)
	}
	return strings.Join(names, ", ")
}

func workspaceContainerNetworkNames(c *docker.Container) string {
	if c == nil || c.Detail.NetworkSettings == nil || len(c.Detail.NetworkSettings.Networks) == 0 {
		return ""
	}
	names := make([]string, 0, len(c.Detail.NetworkSettings.Networks))
	for name := range c.Detail.NetworkSettings.Networks {
		names = append(names, name)
	}
	slices.Sort(names)
	return workspaceJoinLimited(names, 6)
}

func workspaceContainerMountTargets(c *docker.Container) string {
	if c == nil || len(c.Detail.Mounts) == 0 {
		return ""
	}
	targets := make([]string, 0, len(c.Detail.Mounts))
	for _, mount := range c.Detail.Mounts {
		if mount.Destination != "" {
			targets = append(targets, mount.Destination)
		}
	}
	return workspaceJoinLimited(targets, 6)
}

func workspaceDockerStatsLines(stats *dockercontainer.StatsResponse) []string {
	if stats == nil {
		return nil
	}
	var lines []string
	if stats.Name != "" {
		lines = append(lines, fmt.Sprintf("stats.name: %s", strings.TrimPrefix(stats.Name, "/")))
	}
	if stats.ID != "" {
		lines = append(lines, fmt.Sprintf("stats.id: %s", stats.ID))
	}
	if !stats.Read.IsZero() {
		lines = append(lines, fmt.Sprintf("stats.read: %s", stats.Read.Format("2006-01-02 15:04:05")))
	}
	if !stats.PreRead.IsZero() {
		lines = append(lines, fmt.Sprintf("stats.preread: %s", stats.PreRead.Format("2006-01-02 15:04:05")))
	}
	lines = append(lines, workspacePidsStatsLines(stats.PidsStats)...)
	lines = append(lines, workspaceCPUStatsLines("cpu_stats", stats.CPUStats)...)
	lines = append(lines, workspaceCPUStatsLines("precpu_stats", stats.PreCPUStats)...)
	lines = append(lines, workspaceMemoryStatsLines(stats.MemoryStats)...)
	lines = append(lines, workspaceStorageStatsLines(stats.StorageStats)...)
	lines = append(lines, workspaceBlkioStatsLines(stats.BlkioStats)...)
	lines = append(lines, workspaceNetworkStatsLines(stats.Networks)...)
	return lines
}

func workspacePidsStatsLines(stats dockercontainer.PidsStats) []string {
	return []string{
		fmt.Sprintf("pids_stats.current: %d", stats.Current),
		fmt.Sprintf("pids_stats.limit: %d", stats.Limit),
	}
}

func workspaceCPUStatsLines(prefix string, stats dockercontainer.CPUStats) []string {
	lines := []string{
		fmt.Sprintf("%s.cpu_usage.total_usage: %s", prefix, workspaceFormatNanos(stats.CPUUsage.TotalUsage)),
		fmt.Sprintf("%s.cpu_usage.percpu_usage: %s", prefix, workspaceFormatUintSlice(stats.CPUUsage.PercpuUsage, 8)),
		fmt.Sprintf("%s.cpu_usage.usage_in_kernelmode: %s", prefix, workspaceFormatNanos(stats.CPUUsage.UsageInKernelmode)),
		fmt.Sprintf("%s.cpu_usage.usage_in_usermode: %s", prefix, workspaceFormatNanos(stats.CPUUsage.UsageInUsermode)),
		fmt.Sprintf("%s.system_cpu_usage: %s", prefix, workspaceFormatNanos(stats.SystemUsage)),
		fmt.Sprintf("%s.online_cpus: %d", prefix, stats.OnlineCPUs),
		fmt.Sprintf("%s.throttling.periods: %d", prefix, stats.ThrottlingData.Periods),
		fmt.Sprintf("%s.throttling.throttled_periods: %d", prefix, stats.ThrottlingData.ThrottledPeriods),
		fmt.Sprintf("%s.throttling.throttled_time: %s", prefix, workspaceFormatNanos(stats.ThrottlingData.ThrottledTime)),
	}
	return lines
}

func workspaceMemoryStatsLines(stats dockercontainer.MemoryStats) []string {
	lines := []string{
		fmt.Sprintf("memory_stats.usage: %s", workspaceFormatBytesValue(stats.Usage)),
		fmt.Sprintf("memory_stats.max_usage: %s", workspaceFormatBytesValue(stats.MaxUsage)),
		fmt.Sprintf("memory_stats.limit: %s", workspaceFormatBytesValue(stats.Limit)),
		fmt.Sprintf("memory_stats.failcnt: %d", stats.Failcnt),
		fmt.Sprintf("memory_stats.commitbytes: %s", workspaceFormatBytesValue(stats.Commit)),
		fmt.Sprintf("memory_stats.commitpeakbytes: %s", workspaceFormatBytesValue(stats.CommitPeak)),
		fmt.Sprintf("memory_stats.privateworkingset: %s", workspaceFormatBytesValue(stats.PrivateWorkingSet)),
	}
	if len(stats.Stats) > 0 {
		keys := make([]string, 0, len(stats.Stats))
		for key := range stats.Stats {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("memory_stats.stats.%s: %s", key, workspaceFormatBytesValue(stats.Stats[key])))
		}
	}
	return lines
}

func workspaceStorageStatsLines(stats dockercontainer.StorageStats) []string {
	return []string{
		fmt.Sprintf("storage_stats.read_count_normalized: %d", stats.ReadCountNormalized),
		fmt.Sprintf("storage_stats.read_size_bytes: %s", workspaceFormatBytesValue(stats.ReadSizeBytes)),
		fmt.Sprintf("storage_stats.write_count_normalized: %d", stats.WriteCountNormalized),
		fmt.Sprintf("storage_stats.write_size_bytes: %s", workspaceFormatBytesValue(stats.WriteSizeBytes)),
	}
}

func workspaceBlkioStatsLines(stats dockercontainer.BlkioStats) []string {
	var lines []string
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_service_bytes_recursive", stats.IoServiceBytesRecursive, true)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_serviced_recursive", stats.IoServicedRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_queue_recursive", stats.IoQueuedRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_service_time_recursive", stats.IoServiceTimeRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_wait_time_recursive", stats.IoWaitTimeRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_merged_recursive", stats.IoMergedRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.io_time_recursive", stats.IoTimeRecursive, false)...)
	lines = append(lines, workspaceBlkioEntryLines("blkio_stats.sectors_recursive", stats.SectorsRecursive, false)...)
	return lines
}

func workspaceBlkioEntryLines(prefix string, entries []dockercontainer.BlkioStatEntry, bytesValue bool) []string {
	if len(entries) == 0 {
		return []string{fmt.Sprintf("%s: none", prefix)}
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		key := fmt.Sprintf("%s.%d:%d.%s", prefix, entry.Major, entry.Minor, strings.ToLower(entry.Op))
		if bytesValue {
			lines = append(lines, fmt.Sprintf("%s: %s", key, workspaceFormatBytesValue(entry.Value)))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %d", key, entry.Value))
	}
	return lines
}

func workspaceNetworkStatsLines(networks map[string]dockercontainer.NetworkStats) []string {
	if len(networks) == 0 {
		return []string{"networks: none"}
	}
	names := make([]string, 0, len(networks))
	for name := range networks {
		names = append(names, name)
	}
	slices.Sort(names)
	lines := make([]string, 0, len(names)*10)
	for _, name := range names {
		stats := networks[name]
		prefix := fmt.Sprintf("networks.%s", name)
		lines = append(lines,
			fmt.Sprintf("%s.rx_bytes: %s", prefix, workspaceFormatBytesValue(stats.RxBytes)),
			fmt.Sprintf("%s.rx_packets: %d", prefix, stats.RxPackets),
			fmt.Sprintf("%s.rx_errors: %d", prefix, stats.RxErrors),
			fmt.Sprintf("%s.rx_dropped: %d", prefix, stats.RxDropped),
			fmt.Sprintf("%s.tx_bytes: %s", prefix, workspaceFormatBytesValue(stats.TxBytes)),
			fmt.Sprintf("%s.tx_packets: %d", prefix, stats.TxPackets),
			fmt.Sprintf("%s.tx_errors: %d", prefix, stats.TxErrors),
			fmt.Sprintf("%s.tx_dropped: %d", prefix, stats.TxDropped),
		)
		if stats.EndpointID != "" {
			lines = append(lines, fmt.Sprintf("%s.endpoint_id: %s", prefix, stats.EndpointID))
		}
		if stats.InstanceID != "" {
			lines = append(lines, fmt.Sprintf("%s.instance_id: %s", prefix, stats.InstanceID))
		}
	}
	return lines
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

func workspaceJoinLimited(items []string, limit int) string {
	if len(items) == 0 {
		return ""
	}
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	if limit <= 0 || len(filtered) <= limit {
		return strings.Join(filtered, ", ")
	}
	return strings.Join(filtered[:limit], ", ") + fmt.Sprintf(" +%d", len(filtered)-limit)
}

func workspaceFormatSwarmPorts(ports []swarm.PortConfig) string {
	if len(ports) == 0 {
		return ""
	}
	values := make([]string, 0, len(ports))
	for _, port := range ports {
		if port.PublishedPort != 0 {
			values = append(values, fmt.Sprintf("%d->%d/%s", port.PublishedPort, port.TargetPort, port.Protocol))
		} else {
			values = append(values, fmt.Sprintf("%d/%s", port.TargetPort, port.Protocol))
		}
	}
	return workspaceJoinLimited(values, 6)
}

func workspaceFormatTimestamp(value string) string {
	if value == "" || strings.HasPrefix(value, "0001-01-01") {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return t.Format("2006-01-02 15:04")
}

func workspaceFormatUnix(ts int64) string {
	return time.Unix(ts, 0).Format("2006-01-02 15:04")
}

func workspaceFormatBytesValue(v uint64) string {
	return fmt.Sprintf("%d (%s)", v, units.BytesSize(float64(v)))
}

func workspaceFormatNanos(v uint64) string {
	return fmt.Sprintf("%d (%s)", v, time.Duration(v))
}

func workspaceFormatUintSlice(values []uint64, limit int) string {
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, min(len(values), limit))
	for i, value := range values {
		if limit > 0 && i >= limit {
			parts = append(parts, fmt.Sprintf("+%d", len(values)-limit))
			break
		}
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}

func workspaceMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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

func (m model) executeMenuCommand(containerID string, cmd docker.Command) (model, tea.Cmd) {
	switch cmd {
	case docker.INSPECT:
		return m, inspectContainerCmd(m.daemon, containerID)
	case docker.LOGS:
		return m, showContainerLogsCmd(m.daemon, containerID)
	case docker.ATTACH:
		return m, attachContainerCmd(m.daemon, containerID)
	case docker.EXEC:
		var cmd tea.Cmd
		m.inputPrompt, cmd = appui.NewInputPromptModelWithLimit(
			fmt.Sprintf("Exec in %s:", shortID(containerID)),
			"/bin/sh", "exec", containerID, 120,
		)
		m.inputPrompt.SetSize(m.width, m.height)
		m.overlay = overlayInputPrompt
		return m, cmd
	case docker.KILL:
		return m.showPrompt(
			fmt.Sprintf("Kill container %s?", shortID(containerID)),
			"kill", containerID,
		), nil
	case docker.STOP:
		return m.showPrompt(
			fmt.Sprintf("Stop container %s?", shortID(containerID)),
			"stop", containerID,
		), nil
	case docker.RESTART:
		return m.showPrompt(
			fmt.Sprintf("Restart container %s?", shortID(containerID)),
			"restart", containerID,
		), nil
	case docker.RM:
		return m.showPrompt(
			fmt.Sprintf("Remove container %s?", shortID(containerID)),
			"rm", containerID,
		), nil
	case docker.STATS:
		return m, showContainerStatsCmd(m.daemon, containerID)
	case docker.HISTORY:
		if c := m.daemon.ContainerByID(containerID); c != nil {
			return m, showImageHistoryCmd(m.daemon, c.ImageID)
		}
		return m, nil
	}
	return m, nil
}

func (m model) executeContainerOp(tag, id string) tea.Cmd {
	daemon := m.daemon
	return func() tea.Msg {
		var err error
		var successMsg string
		switch tag {
		case "kill":
			err = daemon.Kill(id)
			successMsg = fmt.Sprintf("Container %s killed", shortID(id))
		case "stop":
			err = daemon.StopContainer(id)
			successMsg = fmt.Sprintf("Container %s stopped", shortID(id))
		case "restart":
			err = daemon.RestartContainer(id)
			successMsg = fmt.Sprintf("Container %s restarted", shortID(id))
		case "rm":
			err = daemon.Rm(id)
			successMsg = fmt.Sprintf("Container %s removed", shortID(id))
		case "rm-all-stopped":
			var count int
			count, err = daemon.RemoveAllStoppedContainers()
			successMsg = fmt.Sprintf("Removed %d stopped containers", count)
		case "rmi":
			_, err = daemon.Rmi(id, false)
			successMsg = "Image removed"
		case "rmi-force":
			_, err = daemon.Rmi(id, true)
			successMsg = "Image force removed"
		case "rmi-dangling":
			var count int
			count, err = daemon.RemoveDanglingImages()
			successMsg = fmt.Sprintf("Removed %d dangling images", count)
		case "rmi-unused":
			var count int
			count, err = daemon.RemoveUnusedImages()
			successMsg = fmt.Sprintf("Removed %d unused images", count)
		case "net-rm":
			err = daemon.RemoveNetwork(id)
			successMsg = "Network removed"
		case "vol-rm":
			err = daemon.VolumeRemove(context.Background(), id, false)
			successMsg = fmt.Sprintf("Volume %s removed", id)
		case "vol-rm-force":
			err = daemon.VolumeRemove(context.Background(), id, true)
			successMsg = fmt.Sprintf("Volume %s force removed", id)
		case "vol-rm-all":
			var count int
			count, err = daemon.VolumeRemoveAll(context.Background())
			successMsg = fmt.Sprintf("Removed %d volumes", count)
		case "vol-prune":
			var count int
			count, err = daemon.VolumePrune(context.Background())
			successMsg = fmt.Sprintf("Pruned %d unused volumes", count)
		case "service-rm":
			err = daemon.ServiceRemove(id)
			successMsg = fmt.Sprintf("Service %s removed", shortID(id))
		case "service-update":
			err = daemon.ServiceUpdate(id)
			successMsg = fmt.Sprintf("Service %s update forced", shortID(id))
		case "stack-rm":
			err = daemon.StackRemove(id)
			successMsg = fmt.Sprintf("Stack %s removed", id)
		case "compose-start":
			project, service, _ := strings.Cut(id, "/")
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeServiceStart(project, service)
			successMsg = report.Summary()
		case "compose-stop":
			project, service, _ := strings.Cut(id, "/")
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeServiceStop(project, service)
			successMsg = report.Summary()
		case "compose-restart":
			project, service, _ := strings.Cut(id, "/")
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeServiceRestart(project, service)
			successMsg = report.Summary()
		case "compose-rm":
			project, service, _ := strings.Cut(id, "/")
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeServiceRemove(project, service)
			successMsg = report.Summary()
		case "compose-project-stop":
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeProjectStop(id)
			successMsg = report.Summary()
		case "compose-project-restart":
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeProjectRestart(id)
			successMsg = report.Summary()
		case "compose-project-rm":
			var report docker.ComposeServiceActionReport
			report, err = daemon.ComposeProjectRemove(id)
			successMsg = report.Summary()
		case "prune":
			report, pruneErr := daemon.Prune()
			if pruneErr != nil {
				err = pruneErr
			} else {
				successMsg = fmt.Sprintf("Pruned: %d containers, %d images, %d networks, %d volumes, reclaimed %s",
					len(report.ContainerReport.ContainersDeleted),
					len(report.ImagesReport.ImagesDeleted),
					len(report.NetworksReport.NetworksDeleted),
					len(report.VolumesReport.VolumesDeleted),
					units.BytesSize(float64(report.TotalSpaceReclaimed())))
			}
		default:
			return nil
		}
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return operationSuccessMsg{message: successMsg}
	}
}

func (m model) executeInputOp(tag, id, value string) tea.Cmd {
	daemon := m.daemon
	switch tag {
	case "exec":
		value = strings.TrimSpace(value)
		if value == "" {
			value = "/bin/sh"
		}
		command := strings.Fields(value)
		return execContainerCmd(daemon, id, command)
	case "service-scale":
		var replicas uint64
		if _, err := fmt.Sscanf(value, "%d", &replicas); err != nil {
			return func() tea.Msg {
				return statusMessageMsg{
					text:   fmt.Sprintf("Invalid replica count: %s", value),
					expiry: 5 * time.Second,
				}
			}
		}
		return func() tea.Msg {
			err := daemon.ServiceScale(id, replicas)
			if err != nil {
				return statusMessageMsg{
					text:   fmt.Sprintf("Scale error: %s", err),
					expiry: 5 * time.Second,
				}
			}
			return operationSuccessMsg{
				message: fmt.Sprintf("Service %s scaled to %d replicas", shortID(id), replicas),
			}
		}
	}
	return nil
}

func (m model) cycleNodeAvailability(nodeID string) tea.Cmd {
	daemon := m.daemon
	return func() tea.Msg {
		node, err := daemon.Node(nodeID)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Node error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		// Cycle: active → pause → drain → active
		var next swarm.NodeAvailability
		switch node.Spec.Availability {
		case swarm.NodeAvailabilityActive:
			next = swarm.NodeAvailabilityPause
		case swarm.NodeAvailabilityPause:
			next = swarm.NodeAvailabilityDrain
		default:
			next = swarm.NodeAvailabilityActive
		}
		err = daemon.NodeChangeAvailability(nodeID, next)
		if err != nil {
			return statusMessageMsg{
				text:   fmt.Sprintf("Availability error: %s", err),
				expiry: 5 * time.Second,
			}
		}
		return operationSuccessMsg{
			message: fmt.Sprintf("Node %s availability set to %s", shortID(nodeID), next),
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
