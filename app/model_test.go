package app

import (
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	dockercontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/volume"
	"github.com/moncho/dry/appui"
	appcompose "github.com/moncho/dry/appui/compose"
	appswarm "github.com/moncho/dry/appui/swarm"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/mocks"
)

func newTestModel() model {
	m := NewModel(Config{})
	m.width = 120
	m.height = 40
	m.daemon = &mocks.DockerDaemonMock{}
	m.ready = true
	m.swarmMode = true
	m.containers.SetDaemon(m.daemon)
	m.images.SetDaemon(m.daemon)
	m.networks.SetDaemon(m.daemon)
	m.volumes.SetDaemon(m.daemon)
	ch := m.contentHeight()
	m.containers.SetSize(m.width, ch)
	m.images.SetSize(m.width, ch)
	m.networks.SetSize(m.width, ch)
	m.volumes.SetSize(m.width, ch)
	return m
}

func newWorkspaceTestModel() model {
	m := NewModel(Config{WorkspaceMode: true})
	m.width = 120
	m.height = 40
	m.daemon = &mocks.DockerDaemonMock{}
	m.ready = true
	m.swarmMode = true
	m.containers.SetDaemon(m.daemon)
	m.images.SetDaemon(m.daemon)
	m.networks.SetDaemon(m.daemon)
	m.volumes.SetDaemon(m.daemon)
	m.composeProjects.SetDaemon(m.daemon)
	m.resizeContentModels()
	return m
}

type monitorContainerLookupStub struct {
	container *docker.Container
}

func (s monitorContainerLookupStub) ContainerByID(id string) *docker.Container {
	if s.container != nil && s.container.ID == id {
		return s.container
	}
	return nil
}

func TestModel_ViewSwitching(t *testing.T) {
	m := newTestModel()

	tests := []struct {
		key      string
		expected viewMode
	}{
		{"2", Images},
		{"3", Networks},
		{"4", Volumes},
		{"1", Main},
		{"5", Nodes},
		{"6", Services},
		{"7", Stacks},
	}

	for _, tt := range tests {
		result, _ := m.Update(tea.KeyPressMsg{Code: rune(tt.key[0])})
		m = result.(model)
		if m.view != tt.expected {
			t.Errorf("key %q: expected view %d, got %d", tt.key, tt.expected, m.view)
		}
	}
}

func TestModel_HelpOverlay(t *testing.T) {
	m := newTestModel()

	// Press ? to open help
	result, cmd := m.Update(tea.KeyPressMsg{Code: '?'})
	m = result.(model)

	if cmd == nil {
		t.Fatal("expected a cmd from help key")
	}

	// Execute the cmd — it returns showLessMsg
	msg := cmd()
	if msg == nil {
		t.Fatal("expected non-nil msg from help cmd")
	}

	result, _ = m.Update(msg)
	m = result.(model)

	if m.overlay != overlayLess {
		t.Fatalf("expected overlayLess, got %d", m.overlay)
	}
}

func TestWorkspaceContextFromStatsUsesNameAndPortsWithoutCpuMemory(t *testing.T) {
	rawStats := &dockercontainer.StatsResponse{
		Name: "redis",
		ID:   "abc123",
		Networks: map[string]dockercontainer.NetworkStats{
			"eth0": {
				RxBytes:   2048,
				RxPackets: 12,
				TxBytes:   4096,
				TxPackets: 24,
			},
		},
	}
	rawStats.Read = time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	rawStats.PreRead = time.Date(2026, 3, 22, 9, 59, 59, 0, time.UTC)
	rawStats.PidsStats = dockercontainer.PidsStats{
		Current: 4,
		Limit:   32,
	}
	rawStats.BlkioStats = dockercontainer.BlkioStats{
		IoServiceBytesRecursive: []dockercontainer.BlkioStatEntry{
			{Major: 8, Minor: 0, Op: "Read", Value: 1024},
		},
	}
	rawStats.CPUStats = dockercontainer.CPUStats{
		CPUUsage: dockercontainer.CPUUsage{
			TotalUsage:        1000,
			PercpuUsage:       []uint64{400, 600},
			UsageInKernelmode: 100,
			UsageInUsermode:   900,
		},
		SystemUsage: 2000,
		OnlineCPUs:  8,
		ThrottlingData: dockercontainer.ThrottlingData{
			Periods:          10,
			ThrottledPeriods: 2,
			ThrottledTime:    500,
		},
	}
	rawStats.PreCPUStats = dockercontainer.CPUStats{
		CPUUsage: dockercontainer.CPUUsage{
			TotalUsage: 900,
		},
	}
	rawStats.MemoryStats = dockercontainer.MemoryStats{
		Usage:    2048,
		MaxUsage: 4096,
		Limit:    8192,
		Failcnt:  1,
		Stats: map[string]uint64{
			"cache": 512,
		},
	}

	stats := &docker.Stats{
		CID:              "abc123",
		Command:          "redis-server",
		CPUPercentage:    23.4,
		Memory:           128 * 1024 * 1024,
		MemoryLimit:      1024 * 1024 * 1024,
		MemoryPercentage: 12.5,
		NetworkRx:        2048,
		NetworkTx:        4096,
		Stats:            rawStats,
	}
	lookup := monitorContainerLookupStub{
		container: &docker.Container{
			Summary: dockercontainer.Summary{
				ID:    "abc123",
				Names: []string{"/redis"},
				Ports: []dockercontainer.PortSummary{{PublicPort: 6379, PrivatePort: 6379, Type: "tcp"}},
			},
		},
	}

	ctx := workspaceContextFromStats(stats, lookup, appui.MonitorSeries{
		CPU: []appui.MonitorPoint{
			{Value: 10},
			{Value: 23.4},
		},
		Memory: []appui.MonitorPoint{
			{Value: 5},
			{Value: 12.5},
		},
	})

	if ctx.title != "redis" {
		t.Fatalf("expected monitor context title to use container name, got %q", ctx.title)
	}
	body := strings.Join(ctx.lines, "\n")
	for _, want := range []string{
		"name: redis",
		"ports: 6379->6379/tcp",
		"command: redis-server",
		"stats.name: redis",
		"stats.id: abc123",
		"stats.read: 2026-03-22 10:00:00",
		"pids_stats.limit: 32",
		"cpu_stats.online_cpus: 8",
		"cpu_stats.cpu_usage.percpu_usage: 400, 600",
		"memory_stats.stats.cache: 512 (512B)",
		"blkio_stats.io_service_bytes_recursive.8:0.read: 1024 (1KiB)",
		"networks.eth0.tx_bytes: 4096 (4KiB)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in monitor context, got %q", want, body)
		}
	}
	if strings.Contains(body, "cpu:") || strings.Contains(body, "memory:") {
		t.Fatalf("did not expect cpu/memory summary lines in monitor context, got %q", body)
	}
	if ctx.monitorCPU != 23.4 || ctx.monitorPct != 12.5 {
		t.Fatalf("expected monitor gauge fields to be preserved, got %+v", ctx)
	}
	if len(ctx.monitorCPUHistory) != 2 || len(ctx.monitorMemHistory) != 2 {
		t.Fatalf("expected monitor history to be preserved, got %+v", ctx)
	}
}

func TestModel_CloseOverlay(t *testing.T) {
	m := newTestModel()
	m.overlay = overlayLess

	result, _ := m.Update(appui.CloseOverlayMsg{})
	m = result.(model)

	if m.overlay != overlayNone {
		t.Fatalf("expected overlayNone after CloseOverlayMsg, got %d", m.overlay)
	}
}

func TestModel_PromptConfirm(t *testing.T) {
	m := newTestModel()

	// Show a prompt
	m = m.showPrompt("Test?", "kill", "abc123def456")
	if m.overlay != overlayPrompt {
		t.Fatal("expected overlayPrompt")
	}

	// Confirm — sends PromptResultMsg
	result, cmd := m.Update(tea.KeyPressMsg{Code: 'y'})
	m = result.(model)

	if cmd == nil {
		t.Fatal("expected cmd from prompt confirm")
	}

	// Execute the cmd — should produce PromptResultMsg
	msg := cmd()
	if _, ok := msg.(appui.PromptResultMsg); !ok {
		t.Fatalf("expected PromptResultMsg, got %T", msg)
	}
}

func TestModel_PromptDeny(t *testing.T) {
	m := newTestModel()

	m = m.showPrompt("Test?", "kill", "abc123def456")

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'n'})
	_ = result.(model)

	if cmd == nil {
		t.Fatal("expected cmd from prompt deny")
	}

	msg := cmd()
	pr, ok := msg.(appui.PromptResultMsg)
	if !ok {
		t.Fatalf("expected PromptResultMsg, got %T", msg)
	}
	if pr.Confirmed {
		t.Fatal("expected not confirmed")
	}
}

func TestModel_EscapeGoesToMain(t *testing.T) {
	m := newTestModel()

	// Switch to images
	result, _ := m.Update(tea.KeyPressMsg{Code: '2'})
	m = result.(model)
	if m.view != Images {
		t.Fatalf("expected Images view, got %d", m.view)
	}

	// Press escape — should go to Main
	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = result.(model)
	if m.view != Main {
		t.Fatalf("expected Main view after escape, got %d", m.view)
	}
}

func TestModel_EscapeNoopOnMain(t *testing.T) {
	m := newTestModel()

	if m.view != Main {
		t.Fatalf("expected initial view Main, got %d", m.view)
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = result.(model)
	if m.view != Main {
		t.Fatalf("expected view still Main after escape, got %d", m.view)
	}
}

func TestModel_ContainerMenuOverlay(t *testing.T) {
	m := newTestModel()
	// Load some containers
	containers := m.daemon.Containers(nil, 0)
	m.containers.SetContainers(containers)

	// Press enter — should open container menu
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = result.(model)

	if m.overlay != overlayContainerMenu {
		t.Fatalf("expected overlayContainerMenu, got %d", m.overlay)
	}
}

func TestModel_WorkspaceEnterOpensContainerMenu(t *testing.T) {
	m := newWorkspaceTestModel()
	containers := m.daemon.Containers(nil, 0)
	m.containers.SetContainers(containers)

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = result.(model)

	if m.pinnedContext != nil {
		t.Fatal("expected enter to keep pin state unchanged in workspace main view")
	}
	if m.overlay != overlayContainerMenu {
		t.Fatalf("expected overlayContainerMenu, got %d", m.overlay)
	}
}

func TestModel_WorkspaceLowercasePTogglesPin(t *testing.T) {
	m := newWorkspaceTestModel()
	containers := m.daemon.Containers(nil, 0)
	m.containers.SetContainers(containers)

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'p'})
	m = result.(model)
	if m.pinnedContext == nil {
		t.Fatal("expected pinned context after p")
	}
	if cmd == nil {
		t.Fatal("expected workspace activity load cmd after pin")
	}

	result, cmd = m.Update(tea.KeyPressMsg{Code: 'p'})
	m = result.(model)
	if m.pinnedContext != nil {
		t.Fatal("expected pin to clear after second p")
	}
	if cmd != nil {
		_ = cmd()
	}
}

func TestModel_WorkspaceTabCyclesPane(t *testing.T) {
	m := newWorkspaceTestModel()

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = result.(model)
	if m.activePane != workspacePaneContext {
		t.Fatalf("expected context pane after one tab, got %v", m.activePane)
	}

	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = result.(model)
	if m.activePane != workspacePaneActivity {
		t.Fatalf("expected activity pane after two tabs, got %v", m.activePane)
	}

	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = result.(model)
	if m.activePane != workspacePaneNavigator {
		t.Fatalf("expected navigator pane after three tabs, got %v", m.activePane)
	}
}

func TestModel_WorkspaceShiftTabCyclesPaneInReverse(t *testing.T) {
	m := newWorkspaceTestModel()
	m.activePane = workspacePaneActivity

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m = result.(model)
	if m.activePane != workspacePaneContext {
		t.Fatalf("expected context pane after shift+tab, got %v", m.activePane)
	}

	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m = result.(model)
	if m.activePane != workspacePaneNavigator {
		t.Fatalf("expected navigator pane after second shift+tab, got %v", m.activePane)
	}
}

func TestModel_WorkspaceEscapeClearsPin(t *testing.T) {
	m := newWorkspaceTestModel()
	containers := m.daemon.Containers(nil, 0)
	m.containers.SetContainers(containers)
	ctx := workspaceContextFromContainer(containers[0])
	m.pinnedContext = &ctx

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = result.(model)

	if m.pinnedContext != nil {
		t.Fatal("expected escape to clear workspace pin")
	}
}

func TestModel_WorkspaceContainerViewCapsTopPaneToFiveRows(t *testing.T) {
	m := newWorkspaceTestModel()

	_, _, topH, activityH := m.workspaceLayout()
	if topH != 9 {
		t.Fatalf("expected top pane height 9 in container workspace view, got %d", topH)
	}
	if activityH != m.contentHeight()-1-topH {
		t.Fatalf("expected remaining height to be assigned to activity, got %d", activityH)
	}
}

func TestModel_WorkspaceMonitorViewKeepsExtraF7SpaceForActivity(t *testing.T) {
	m := newWorkspaceTestModel()
	m.view = Monitor
	m.resizeContentModels()

	_, _, topBefore, activityBefore := m.workspaceLayout()
	if topBefore != 5 {
		t.Fatalf("expected empty monitor workspace top pane height 5, got %d", topBefore)
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	m = result.(model)

	_, _, topAfter, activityAfter := m.workspaceLayout()
	if topAfter != topBefore {
		t.Fatalf("expected monitor top pane height to stay capped after F7, got %d -> %d", topBefore, topAfter)
	}
	if activityAfter <= activityBefore {
		t.Fatalf("expected extra height after F7 to go to activity pane, got %d -> %d", activityBefore, activityAfter)
	}
}

func TestModel_WorkspaceMonitorViewShrinksToVisibleRowCount(t *testing.T) {
	m := newWorkspaceTestModel()
	m.view = Monitor
	m.resizeContentModels()

	ch := make(chan *docker.Stats)
	m.monitor.UpdateStats("aaa", &docker.Stats{CID: "aaa", CPUPercentage: 10, MemoryPercentage: 20}, ch)
	m.monitor.UpdateStats("bbb", &docker.Stats{CID: "bbb", CPUPercentage: 30, MemoryPercentage: 40}, ch)
	m.monitor.FlushTable()

	_, _, topH, _ := m.workspaceLayout()
	if topH != 6 {
		t.Fatalf("expected monitor workspace top pane height 6 for two rows, got %d", topH)
	}
}

func TestModel_WorkspaceTabsReflectActivePane(t *testing.T) {
	m := newWorkspaceTestModel()

	navigatorTabs := m.renderWorkspaceTabs()
	if !strings.Contains(navigatorTabs, "Navigator") || !strings.Contains(navigatorTabs, "Activity") || !strings.Contains(navigatorTabs, "Context") {
		t.Fatal("expected workspace tabs to render navigator, context, and activity labels")
	}

	m.activePane = workspacePaneActivity
	activityTabs := m.renderWorkspaceTabs()
	if navigatorTabs == activityTabs {
		t.Fatal("expected active tab styling to change when pane focus changes")
	}
}

func TestModel_WorkspaceFooterShowsOnlyWorkspaceControls(t *testing.T) {
	m := newWorkspaceTestModel()

	footer := m.renderFooter()
	if !strings.Contains(footer, "1-8/m") || !strings.Contains(footer, "nav") {
		t.Fatal("expected navigation binding in workspace footer")
	}
	if !strings.Contains(footer, "spc") || !strings.Contains(footer, "peek") {
		t.Fatal("expected quick peek binding in workspace footer")
	}
	if !strings.Contains(footer, "tab") || !strings.Contains(footer, "pane") {
		t.Fatal("expected pane navigation binding in workspace footer")
	}
	if !strings.Contains(footer, "↑↓") || !strings.Contains(footer, "move") {
		t.Fatal("expected movement binding in workspace footer")
	}
	if !strings.Contains(footer, "pin") {
		t.Fatal("expected pin binding in workspace footer")
	}
	if !strings.Contains(footer, "F1") || !strings.Contains(footer, "F2") || !strings.Contains(footer, "F5") {
		t.Fatal("expected container function bindings in workspace footer")
	}
	if !strings.Contains(footer, "↵") || !strings.Contains(footer, "open") {
		t.Fatal("expected enter binding in workspace footer")
	}
	if !strings.Contains(footer, "?") {
		t.Fatal("expected help binding in workspace footer")
	}
	if strings.Contains(footer, "palette") || strings.Contains(footer, "theme") || strings.Contains(footer, "logs") || strings.Contains(footer, "stats") {
		t.Fatal("did not expect non-workspace action bindings in workspace footer")
	}
}

func TestModel_WorkspaceFooterCompactsOnNarrowTerminals(t *testing.T) {
	m := newWorkspaceTestModel()
	m.width = 80

	footer := m.renderFooter()
	for _, want := range []string{"1-8/m", "tab", "p", "spc", "↑↓", "↵", "F1", "F2", "F5", "?"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("expected %q in compact workspace footer, got %q", want, footer)
		}
	}
}

func TestModel_WorkspacePinnedStateAppearsInContextHeader(t *testing.T) {
	m := newWorkspaceTestModel()
	ctx := workspaceContextFromContainer(m.daemon.Containers(nil, 0)[0])
	m.pinnedContext = &ctx

	body := m.renderWorkspaceBody()
	if !strings.Contains(body, "Context · pinned") {
		t.Fatalf("expected pinned context header, got %q", body)
	}
}

func TestModel_WorkspaceContextPaneScrollsWhenFocused(t *testing.T) {
	m := newWorkspaceTestModel()
	m.width = 140
	m.height = 28
	m.resizeContentModels()
	lines := make([]string, 0, 40)
	for i := 1; i <= 40; i++ {
		lines = append(lines, fmt.Sprintf("line %d: value", i))
	}
	m.pinnedContext = &workspaceContext{
		title:    "overflow",
		subtitle: "Context",
		lines:    lines,
	}

	_ = m.renderWorkspaceBody()
	m.activePane = workspacePaneContext
	before := m.workspaceContext.View()

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = result.(model)
	after := m.workspaceContext.View()

	if before == after {
		t.Fatal("expected context pane view to change after scrolling down")
	}
}

func TestWorkspaceContextFromContainerIncludesRicherSummary(t *testing.T) {
	c := &docker.Container{
		Summary: dockercontainer.Summary{
			ID:      "0123456789abcdef",
			Image:   "nginx:latest",
			Status:  "Up 5 minutes",
			Labels:  map[string]string{"app": "api", "tier": "web"},
			Created: 1710000000,
			Ports: []dockercontainer.PortSummary{
				{PublicPort: 8080, PrivatePort: 80, Type: "tcp"},
			},
		},
		Detail: dockercontainer.InspectResponse{
			State: &dockercontainer.State{
				Status: "running",
				Health: &dockercontainer.Health{
					Status: "healthy",
				},
				StartedAt: "2026-03-20T12:00:00Z",
			},
			RestartCount: 2,
			Config: &dockercontainer.Config{
				WorkingDir: "/srv/app",
				User:       "app",
				Env:        []string{"A=1", "B=2"},
			},
			NetworkSettings: &dockercontainer.NetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"bridge": {},
					"edge":   {},
				},
			},
			Mounts: []dockercontainer.MountPoint{{Destination: "/data"}},
		},
	}

	ctx := workspaceContextFromContainer(c)
	summary := strings.Join(ctx.lines, "\n")
	for _, want := range []string{
		"id: 0123456789ab",
		"state: running",
		"health: healthy",
		"restarts: 2",
		"user: app",
		"workdir: /srv/app",
		"env: 2 vars",
		"ports: 8080->80/tcp",
		"mounts: 1",
		"mount targets: /data",
		"networks: 2",
		"network names: bridge, edge",
		"labels: 2",
		"label keys: app, tier",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("expected %q in container summary: %s", want, summary)
		}
	}
}

func TestWorkspaceContextFromNetworkIncludesAttachedEndpoints(t *testing.T) {
	ctx := workspaceContextFromNetwork(network.Inspect{
		Network: network.Network{
			ID:         "abc123456789",
			Name:       "app-net",
			Driver:     "bridge",
			Scope:      "local",
			Attachable: true,
			Internal:   true,
			Labels:     map[string]string{"project": "api"},
			IPAM: network.IPAM{
				Driver: "default",
				Config: []network.IPAMConfig{{
					Subnet:  netip.MustParsePrefix("172.20.0.0/16"),
					Gateway: netip.MustParseAddr("172.20.0.1"),
				}},
			},
		},
		Containers: map[string]network.EndpointResource{
			"1": {Name: "api-1"},
			"2": {Name: "worker-1"},
		},
	})

	summary := strings.Join(ctx.lines, "\n")
	for _, want := range []string{
		"gateway: 172.20.0.1",
		"internal: true",
		"attachable: true",
		"labels: 1",
		"label keys: project",
		"attached: api-1, worker-1",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("expected %q in network summary: %s", want, summary)
		}
	}
}

func TestWorkspaceContextFromVolumeIncludesOptionsAndCreatedAt(t *testing.T) {
	ctx := workspaceContextFromVolume(&volume.Volume{
		Name:       "pgdata",
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/pgdata/_data",
		CreatedAt:  "2026-03-20T18:12:00Z",
		Scope:      "local",
		Labels:     map[string]string{"project": "api"},
		Options:    map[string]string{"device": "/tmp/disk"},
	})

	summary := strings.Join(ctx.lines, "\n")
	for _, want := range []string{
		"name: pgdata",
		"created: 2026-03-20T18:12:00Z",
		"labels: 1",
		"label keys: project",
		"options: 1",
		"option keys: device",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("expected %q in volume summary: %s", want, summary)
		}
	}
}

func TestWorkspaceContextFromSwarmServiceIncludesOperationalSummary(t *testing.T) {
	replicas := uint64(3)
	service := swarm.Service{
		ID: "svc123456789",
		Spec: swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name:   "api",
				Labels: map[string]string{"stack": "payments"},
			},
			Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image:   "nginx:latest",
					Command: []string{"nginx", "-g", "daemon off;"},
					Env:     []string{"A=1"},
				},
				Networks: []swarm.NetworkAttachmentConfig{{Target: "frontend"}},
			},
			UpdateConfig:   &swarm.UpdateConfig{FailureAction: "pause"},
			RollbackConfig: &swarm.UpdateConfig{FailureAction: "rollback"},
		},
		Endpoint: swarm.Endpoint{
			Ports: []swarm.PortConfig{{PublishedPort: 8080, TargetPort: 80, Protocol: network.TCP}},
		},
		ServiceStatus: &swarm.ServiceStatus{RunningTasks: 2, DesiredTasks: 3},
		UpdateStatus:  &swarm.UpdateStatus{State: swarm.UpdateStateUpdating, Message: "rolling"},
	}

	ctx := workspaceContextFromSwarmService(service)
	summary := strings.Join(ctx.lines, "\n")
	for _, want := range []string{
		"tasks: 2/3 running",
		"image: nginx:latest",
		"command: nginx, -g, daemon off;",
		"ports: 8080->80/tcp",
		"networks: 1",
		"update: updating",
		"update policy: pause",
		"rollback policy: rollback",
		"labels: 1",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("expected %q in service summary: %s", want, summary)
		}
	}
}

func TestModel_WorkspacePreviewShowsImageSelection(t *testing.T) {
	m := newWorkspaceTestModel()
	m.view = Images
	imgs, err := m.daemon.Images()
	if err != nil {
		t.Fatalf("unexpected images error: %v", err)
	}
	m.images.SetImages(imgs)

	ctx, ok := m.currentWorkspacePreview()
	if !ok {
		t.Fatal("expected image preview context")
	}
	if ctx.subtitle != "Image" {
		t.Fatalf("expected image subtitle, got %q", ctx.subtitle)
	}
	if len(ctx.lines) == 0 {
		t.Fatal("expected image context lines")
	}
}

func TestModel_WorkspaceImagesLoadInspectIntoActivity(t *testing.T) {
	m := newWorkspaceTestModel()
	m.view = Images
	imgs, err := m.daemon.Images()
	if err != nil {
		t.Fatalf("unexpected images error: %v", err)
	}

	result, cmd := m.Update(appui.ImagesLoadedMsg{Images: imgs})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected workspace activity cmd for image inspect")
	}

	msg := cmd()
	inspectMsg, ok := msg.(workspaceActivityLoadedMsg)
	if !ok {
		t.Fatalf("expected workspaceActivityLoadedMsg, got %T", msg)
	}
	if !strings.Contains(inspectMsg.title, "Image Inspect") {
		t.Fatalf("expected image inspect title, got %q", inspectMsg.title)
	}
}

func TestModel_WorkspaceNodesLoadInspectIntoActivity(t *testing.T) {
	m := newWorkspaceTestModel()
	m.view = Nodes

	node := swarm.Node{}
	node.ID = "node-1"
	node.Description.Hostname = "manager-1"

	result, cmd := m.Update(appswarm.NodesLoadedMsg{Nodes: []swarm.Node{node}})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected workspace activity cmd for node inspect")
	}

	msg := cmd()
	inspectMsg, ok := msg.(workspaceActivityLoadedMsg)
	if !ok {
		t.Fatalf("expected workspaceActivityLoadedMsg, got %T", msg)
	}
	if !strings.Contains(inspectMsg.title, "Node Inspect") {
		t.Fatalf("expected node inspect title, got %q", inspectMsg.title)
	}
}

func TestModel_WorkspaceStacksLoadDetailsIntoActivity(t *testing.T) {
	m := newWorkspaceTestModel()
	m.view = Stacks

	result, cmd := m.Update(appswarm.StacksLoadedMsg{Stacks: []docker.Stack{{Name: "payments", Services: 2}}})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected workspace activity cmd for stack details")
	}

	msg := cmd()
	detailsMsg, ok := msg.(workspaceActivityLoadedMsg)
	if !ok {
		t.Fatalf("expected workspaceActivityLoadedMsg, got %T", msg)
	}
	if !strings.Contains(detailsMsg.title, "Stack Details") {
		t.Fatalf("expected stack details title, got %q", detailsMsg.title)
	}
}

func TestModel_WorkspaceTasksLoadInspectIntoActivity(t *testing.T) {
	m := newWorkspaceTestModel()
	m.view = Tasks

	task := swarm.Task{}
	task.ID = "task-1"

	result, cmd := m.Update(appswarm.TasksLoadedMsg{Tasks: []swarm.Task{task}, Title: "Tasks"})
	m = result.(model)
	if cmd == nil {
		t.Fatal("expected workspace activity cmd for task inspect")
	}

	msg := cmd()
	inspectMsg, ok := msg.(workspaceActivityLoadedMsg)
	if !ok {
		t.Fatalf("expected workspaceActivityLoadedMsg, got %T", msg)
	}
	if !strings.Contains(inspectMsg.title, "Task Inspect") {
		t.Fatalf("expected task inspect title, got %q", inspectMsg.title)
	}
}

func TestModel_WorkspaceComposeServicesEnterKeepsPrimaryInspectAction(t *testing.T) {
	m := newWorkspaceTestModel()
	m.view = ComposeServices
	m.selectedProject = "webapp"

	services := m.daemon.ComposeServices("webapp")
	result, _ := m.Update(appcompose.ServicesLoadedMsg{
		Services: services,
		Project:  "webapp",
	})
	m = result.(model)

	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = result.(model)

	result, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = result.(model)

	if m.pinnedContext != nil {
		t.Fatal("expected enter to preserve inspect behavior in compose services workspace view")
	}
	if cmd == nil {
		t.Fatal("expected inspect command from enter in compose services workspace view")
	}
}

func TestModel_ColonOpensCommandPalette(t *testing.T) {
	m := newTestModel()

	result, cmd := m.Update(tea.KeyPressMsg{Code: ':'})
	m = result.(model)

	if m.overlay != overlayCommandPalette {
		t.Fatalf("expected overlayCommandPalette, got %d", m.overlay)
	}
	if cmd == nil {
		t.Fatal("expected focus cmd when opening command palette")
	}
}

func TestModel_SpaceOpensQuickPeekForImageSelection(t *testing.T) {
	m := newWorkspaceTestModel()
	m.view = Images
	m.images.SetImages([]image.Summary{{
		ID:          "sha256:0123456789abcdef0123456789abcdef",
		RepoTags:    []string{"example/api:latest"},
		RepoDigests: []string{"example/api@sha256:0123456789abcdef"},
		Created:     time.Now().Unix(),
		Size:        12345,
	}})

	result, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = result.(model)

	if m.overlay != overlayQuickPeek {
		t.Fatalf("expected overlayQuickPeek, got %d", m.overlay)
	}
	if cmd == nil {
		t.Fatal("expected quick peek load cmd")
	}

	msg := cmd()
	loaded, ok := msg.(quickPeekLoadedMsg)
	if !ok {
		t.Fatalf("expected quickPeekLoadedMsg, got %T", msg)
	}
	if loaded.title != "example/api:latest" {
		t.Fatalf("expected image quick peek title, got %q", loaded.title)
	}

	result, _ = m.Update(msg)
	m = result.(model)
	view := m.quickPeek.View()
	if !strings.Contains(view, "Quick Peek") {
		t.Fatal("expected quick peek overlay title")
	}
	if !strings.Contains(view, "example/api:latest") {
		t.Fatal("expected selected image title in quick peek")
	}
}

func TestModel_CommandPaletteSwitchActionChangesView(t *testing.T) {
	m := newTestModel()
	m.overlay = overlayCommandPalette

	result, cmd := m.Update(appui.CommandPaletteResultMsg{ActionID: "switch:images"})
	m = result.(model)

	if m.overlay != overlayNone {
		t.Fatalf("expected palette overlay to close, got %d", m.overlay)
	}
	if m.view != Images {
		t.Fatalf("expected Images view after palette action, got %d", m.view)
	}
	if cmd == nil {
		t.Fatal("expected switch view load cmd from palette action")
	}
}

func TestModel_CommandPaletteIncludesContainerActionsForSelection(t *testing.T) {
	m := newTestModel()
	containers := m.daemon.Containers(nil, 0)
	m.containers.SetContainers(containers)

	actions := m.commandPaletteActions()
	foundLogs := false
	foundMenu := false
	for _, action := range actions {
		if action.ID == "container:logs" {
			foundLogs = true
		}
		if action.ID == "container:commands" {
			foundMenu = true
		}
	}
	if !foundLogs || !foundMenu {
		t.Fatalf("expected container palette actions, logs=%v menu=%v", foundLogs, foundMenu)
	}
}

func TestModel_CommandPaletteIncludesGlobalPrune(t *testing.T) {
	m := newTestModel()

	actions := m.commandPaletteActions()
	for _, action := range actions {
		if action.ID == "global:prune" {
			return
		}
	}
	t.Fatal("expected global prune action in command palette")
}

func TestModel_WorkspaceCommandPaletteIncludesFullOpenActions(t *testing.T) {
	m := newWorkspaceTestModel()
	containers := m.daemon.Containers(nil, 0)
	m.containers.SetContainers(containers)

	actions := m.commandPaletteActions()
	foundInspect := false
	foundLogs := false
	for _, action := range actions {
		if action.ID == "workspace:open-inspect" {
			foundInspect = true
		}
		if action.ID == "workspace:open-logs" {
			foundLogs = true
		}
	}
	if !foundInspect || !foundLogs {
		t.Fatalf("expected workspace full-open actions, inspect=%v logs=%v", foundInspect, foundLogs)
	}
}

func TestModel_WorkspaceContextEmptyMessageTracksView(t *testing.T) {
	m := newWorkspaceTestModel()
	m.view = Images
	m.images.SetImages(nil)

	body := m.renderWorkspaceBody()
	if !strings.Contains(body, "Select an image to preview it here.") {
		t.Fatalf("expected image-specific empty message, got %q", body)
	}
	if strings.Contains(body, "Select a container or Compose resource to preview it here.") {
		t.Fatal("did not expect the old generic container/Compose empty message")
	}
}

func TestModel_WorkspaceSwitchViewResetsActivity(t *testing.T) {
	m := newWorkspaceTestModel()
	m.workspaceLogs.SetContent("Activity", "Test status", "stale content")

	result, cmd := m.switchView(Images)
	m = result.(model)

	if cmd == nil {
		t.Fatal("expected load command when switching to images")
	}
	if !strings.Contains(m.workspaceLogs.View(), "Select an image to inspect it here.") {
		t.Fatal("expected workspace activity to reset to the target view placeholder")
	}
	if strings.Contains(m.workspaceLogs.View(), "stale content") {
		t.Fatal("did not expect stale activity content after view switch")
	}
}

func TestModel_ExecuteMenuCommandAttach(t *testing.T) {
	m := newTestModel()
	_, cmd := m.executeMenuCommand("abc123def456", docker.ATTACH)
	if cmd == nil {
		t.Fatal("expected attach command to return non-nil cmd")
	}
}

func TestModel_F7TogglesHeader(t *testing.T) {
	m := newTestModel()

	initialHeader := m.showHeader
	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	m = result.(model)

	if m.showHeader == initialHeader {
		t.Fatal("expected header toggle")
	}

	result, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF7})
	m = result.(model)

	if m.showHeader != initialHeader {
		t.Fatal("expected header toggled back")
	}
}

func TestModel_ContainersLoadedMsg(t *testing.T) {
	m := newTestModel()
	containers := m.daemon.Containers(nil, 0)

	result, _ := m.Update(containersLoadedMsg{containers: containers})
	m = result.(model)

	if m.containers.SelectedContainer() == nil {
		t.Fatal("expected containers to be loaded and selectable")
	}
}

func TestModel_OperationSuccessMsg(t *testing.T) {
	m := newTestModel()

	result, cmd := m.Update(operationSuccessMsg{message: "done!"})
	_ = result.(model)

	// Should trigger a reload for the current view
	if cmd == nil {
		t.Fatal("expected cmd from operationSuccessMsg")
	}
}

func TestModel_StatusMessageMsg(t *testing.T) {
	m := newTestModel()

	result, _ := m.Update(statusMessageMsg{text: "test message"})
	_ = result.(model)
	// No crash is sufficient — message bar state is internal
}

func TestModel_ShortID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc123def456789", "abc123def456"},
		{"short", "short"},
		{"exactly12ch", "exactly12ch"},
		{"", ""},
	}

	for _, tt := range tests {
		got := shortID(tt.input)
		if got != tt.expected {
			t.Errorf("shortID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestModel_LessScrolling(t *testing.T) {
	m := newTestModel()

	// Create a showLessMsg with many lines
	lines := make([]string, 100)
	for i := range 100 {
		lines[i] = fmt.Sprintf("Line %d: content", i+1)
	}
	content := strings.Join(lines, "\n")

	result, _ := m.Update(showLessMsg{content: content, title: "Test"})
	m = result.(model)

	if m.overlay != overlayLess {
		t.Fatalf("expected overlayLess, got %d", m.overlay)
	}

	v1 := m.View()

	// Scroll down with 'j'
	result, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m = result.(model)
	result, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m = result.(model)
	result, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m = result.(model)

	v2 := m.View()

	if v1.Content == v2.Content {
		t.Fatal("expected view to change after scrolling in less overlay")
	}
}

func TestModel_ResizeWithOverlay(t *testing.T) {
	m := newTestModel()

	// Open less overlay
	lines := make([]string, 50)
	for i := range 50 {
		lines[i] = fmt.Sprintf("Line %d", i+1)
	}
	result, _ := m.Update(showLessMsg{content: strings.Join(lines, "\n"), title: "Test"})
	m = result.(model)

	if m.overlay != overlayLess {
		t.Fatalf("expected overlayLess, got %d", m.overlay)
	}

	v1 := m.View()

	// Resize terminal
	result, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = result.(model)

	if m.width != 60 || m.height != 20 {
		t.Fatalf("expected 60x20, got %dx%d", m.width, m.height)
	}

	v2 := m.View()
	if v1.Content == v2.Content {
		t.Fatal("expected view to change after resize with less overlay active")
	}
}
