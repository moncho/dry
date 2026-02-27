package app

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/go-units"
	"github.com/moncho/dry/appui"
	appswarm "github.com/moncho/dry/appui/swarm"
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
	daemon       docker.ContainerDaemon
	config       Config
	swarmMode    bool
	eventsChan   <-chan events.Message
	eventsCancel context.CancelFunc

	// Sub-models
	containers appui.ContainersModel
	images     appui.ImagesModel
	networks   appui.NetworksModel
	volumes    appui.VolumesModel
	diskUsage  appui.DiskUsageModel
	monitor    appui.MonitorModel
	nodes      appswarm.NodesModel
	services   appswarm.ServicesModel
	stacks     appswarm.StacksModel
	tasks      appswarm.TasksModel
	header     appui.HeaderModel
	messageBar appui.MessageBarModel

	// Overlay state
	overlay       overlayType
	less          appui.LessModel
	prompt        appui.PromptModel
	inputPrompt   appui.InputPromptModel
	containerMenu appui.ContainerMenuModel
	streamReader  io.ReadCloser // active streaming reader (logs)
	eventsLive    bool          // true when events less overlay is open

	// Docker event throttling
	pendingRefresh map[docker.SourceType]bool
	refreshTimer   bool

	// Loading animation
	loadingFrame int
	loadingFwd   bool
}

// NewModel creates a new top-level model.
func NewModel(cfg Config) model {
	return model{
		config:         cfg,
		view:           Main,
		showHeader:     true,
		containers:     appui.NewContainersModel(),
		images:         appui.NewImagesModel(),
		networks:       appui.NewNetworksModel(),
		volumes:        appui.NewVolumesModel(),
		diskUsage:      appui.NewDiskUsageModel(),
		monitor:        appui.NewMonitorModel(),
		nodes:          appswarm.NewNodesModel(),
		services:       appswarm.NewServicesModel(),
		stacks:         appswarm.NewStacksModel(),
		tasks:          appswarm.NewTasksModel(),
		pendingRefresh: make(map[docker.SourceType]bool),
		loadingFwd:     true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		connectToDockerCmd(m.config),
		loadingTickCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		ch := m.contentHeight()
		m.containers.SetSize(m.width, ch)
		m.images.SetSize(m.width, ch)
		m.networks.SetSize(m.width, ch)
		m.volumes.SetSize(m.width, ch)
		m.diskUsage.SetSize(m.width, ch)
		m.monitor.SetSize(m.width, ch)
		m.nodes.SetSize(m.width, ch)
		m.services.SetSize(m.width, ch)
		m.stacks.SetSize(m.width, ch)
		m.tasks.SetSize(m.width, ch)
		m.header.SetWidth(m.width)
		// Update overlay sizes
		m.less.SetSize(m.width, m.height)
		m.prompt.SetWidth(m.width)
		m.inputPrompt.SetWidth(m.width)
		m.containerMenu.SetSize(m.width, m.height)
		return m, nil

	case dockerConnectedMsg:
		m.daemon = msg.daemon
		m.ready = true
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
		return m, tea.Batch(
			loadContainersCmd(m.daemon, m.containers.ShowAll(), m.containers.SortMode()),
			listenDockerEvents(m.eventsChan),
		)

	case dockerErrorMsg:
		// Fatal error — can't connect to Docker
		m.messageBar.SetMessage(fmt.Sprintf("Error: %s", msg.err), 10*time.Second)
		return m, tea.Quit

	case containersLoadedMsg:
		m.containers.SetContainers(msg.containers)
		return m, nil

	case appui.ImagesLoadedMsg:
		m.images.SetImages(msg.Images)
		return m, nil

	case appui.NetworksLoadedMsg:
		m.networks.SetNetworks(msg.Networks)
		return m, nil

	case appui.VolumesLoadedMsg:
		m.volumes.SetVolumes(msg.Volumes)
		return m, nil

	case appui.DiskUsageLoadedMsg:
		m.diskUsage.SetUsage(msg.Usage)
		return m, nil

	case appui.MonitorStatsMsg:
		cmd := m.monitor.UpdateStats(msg.CID, msg.Stats, msg.StatsCh)
		return m, cmd

	case appui.MonitorErrorMsg:
		m.monitor.RemoveContainer(msg.CID)
		return m, nil

	case appswarm.NodesLoadedMsg:
		m.nodes.SetNodes(msg.Nodes)
		return m, nil

	case appswarm.ServicesLoadedMsg:
		m.services.SetServices(msg.Services)
		return m, nil

	case appswarm.StacksLoadedMsg:
		m.stacks.SetStacks(msg.Stacks)
		return m, nil

	case appswarm.TasksLoadedMsg:
		m.tasks.SetTasks(msg.Tasks, msg.Title)
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
		// Forward mouse wheel events to overlays for scrolling
		if m.overlay == overlayLess {
			var cmd tea.Cmd
			m.less, cmd = m.less.Update(msg)
			return m, cmd
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
		return m, tea.Quit
	}

	// When a sub-model's filter input is active, delegate directly
	// so keys like Esc, numbers, etc. go to the filter, not global handlers.
	if m.filterActive() {
		return m.delegateToSubModel(msg)
	}

	// Global keys
	switch msg.String() {
	case "f7":
		m.showHeader = !m.showHeader
		ch := m.contentHeight()
		m.containers.SetSize(m.width, ch)
		m.images.SetSize(m.width, ch)
		m.networks.SetSize(m.width, ch)
		m.volumes.SetSize(m.width, ch)
		m.diskUsage.SetSize(m.width, ch)
		m.monitor.SetSize(m.width, ch)
		m.nodes.SetSize(m.width, ch)
		m.services.SetSize(m.width, ch)
		m.stacks.SetSize(m.width, ch)
		m.tasks.SetSize(m.width, ch)
		return m, nil
	case "ctrl+0":
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
	case "esc":
		// Escape goes back to main from any non-main, non-task view
		if m.view != Main && m.view != ServiceTasks && m.view != Tasks && m.view != StackTasks {
			return m.switchView(Main)
		}
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
		return m, cmd

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
		return m, cmd

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
		return m, cmd

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
		return m, cmd

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
		return m, cmd

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
					m.inputPrompt.SetWidth(m.width)
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
		return m, cmd

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
		return m, cmd

	case ServiceTasks, Tasks, StackTasks:
		switch msg.String() {
		case "esc":
			m.view = m.previousView
			return m, m.loadViewData(m.view)
		}
		var cmd tea.Cmd
		m.tasks, cmd = m.tasks.Update(msg)
		return m, cmd
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
		content = m.renderMainScreenWithFooter(m.inputPrompt.View())
	} else if m.overlay == overlayContainerMenu {
		content = m.containerMenu.View()
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
	default:
		sections = append(sections, "View not yet implemented")
	}

	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m model) renderFooter() string {
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
	default:
		bindings = containerKeys.ShortHelp()
	}
	bindings = append(bindings, globalKeys.Theme)

	footerBg := lipgloss.NewStyle().Background(appui.DryTheme.Footer)
	keyStyle := lipgloss.NewStyle().Foreground(appui.DryTheme.Key).Background(appui.DryTheme.Footer)
	descStyle := lipgloss.NewStyle().Foreground(appui.DryTheme.FgSubtle).Background(appui.DryTheme.Footer)
	sepStyle := lipgloss.NewStyle().Foreground(appui.DryTheme.FgSubtle).Background(appui.DryTheme.Footer)

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

	line := b.String()
	w := ansi.StringWidth(line)
	if w > m.width {
		line = ansi.Truncate(line, m.width, "")
	} else if w < m.width {
		line += footerBg.Render(strings.Repeat(" ", m.width-w))
	}
	return line
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

	whale := ui.Cyan(frame)
	connectLine := ui.White(connecting)
	verLine := ui.Blue("Dry Version: ") + ui.White(version.VERSION)

	var inner string
	if m.config.DockerHost != "" {
		hostLine := ui.Blue("Docker Host: ") + ui.White(m.config.DockerHost)
		inner = lipgloss.JoinVertical(lipgloss.Center,
			connectLine, "", whale, "", verLine, hostLine)
	} else {
		inner = lipgloss.JoinVertical(lipgloss.Center,
			connectLine, "", whale, "", verLine)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(appui.DryTheme.Border).
		Padding(1, 3)

	content := box.Render(inner)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
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
	}
	return nil
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
