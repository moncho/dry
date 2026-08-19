package app

// Characterization snapshots for every view, rendered with the mock daemon
// at fixed sizes. They exist to make refactors of app/model.go and the
// docker interfaces provably behavior-preserving: a refactor PR must keep
// these files byte-identical. When a rendering change is intentional,
// regenerate with:
//
//	go test ./app -run TestGoldenViews -update
//
// and review the golden diffs like code.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/mocks"
)

var updateGolden = flag.Bool("update", false, "rewrite golden view files")

// goldenDurationRe matches the relative durations rendered by
// units.HumanDuration ("13 years ago", "About a minute", "Less than a
// second"). They change as wall-clock time advances, so they are normalized
// out of the snapshots.
var goldenDurationRe = regexp.MustCompile(`(?i)\b(?:about )?(?:less than )?(?:an?|\d+) (?:second|minute|hour|day|week|month|year)s?(?: ago)?\b`)

func normalizeGoldenView(view string) string {
	return goldenDurationRe.ReplaceAllString(view, "«DURATION»")
}

func goldenStats() []*docker.Stats {
	return []*docker.Stats{
		{
			CID:              "aaaaaaaaaaaa",
			ID:               strings.Repeat("a", 64),
			Name:             "api",
			Command:          "api-server",
			CPUPercentage:    12.5,
			Memory:           512 * 1024 * 1024,
			MemoryLimit:      2 * 1024 * 1024 * 1024,
			MemoryPercentage: 25.0,
			NetworkRx:        120 * 1024 * 1024,
			NetworkTx:        8 * 1024 * 1024,
			BlockRead:        480 * 1024 * 1024,
			BlockWrite:       32 * 1024 * 1024,
			PidsCurrent:      12,
		},
		{
			CID:              "bbbbbbbbbbbb",
			ID:               strings.Repeat("b", 64),
			Name:             "worker",
			Command:          "worker",
			CPUPercentage:    82.1,
			Memory:           1024 * 1024 * 1024,
			MemoryLimit:      2 * 1024 * 1024 * 1024,
			MemoryPercentage: 50.0,
			NetworkRx:        10 * 1024 * 1024,
			NetworkTx:        90 * 1024 * 1024,
			BlockRead:        64 * 1024 * 1024,
			BlockWrite:       256 * 1024 * 1024,
			PidsCurrent:      4,
		},
	}
}

func goldenTasks() []swarm.Task {
	// Anchored to now so the rendered relative duration ("30 minutes ago")
	// is identical on every run; an absolute date would advance with the
	// wall clock, and cell truncation can cut the duration mid-token where
	// the normalizer cannot catch it.
	timestamp := time.Now().Add(-30 * time.Minute)
	return []swarm.Task{
		{
			ID:           "task1111111111",
			ServiceID:    "service111111",
			NodeID:       "node11111111",
			Slot:         1,
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning, Timestamp: timestamp},
		},
		{
			ID:           "task2222222222",
			ServiceID:    "service222222",
			NodeID:       "node11111111",
			Slot:         2,
			DesiredState: swarm.TaskStateShutdown,
			Status:       swarm.TaskStatus{State: swarm.TaskStateFailed, Timestamp: timestamp, Err: "task: non-zero exit (1)"},
		},
	}
}

func newGoldenModel(t *testing.T, workspace bool, width, height int) model {
	t.Helper()
	appui.InitStyles()

	cfg := Config{}
	if workspace {
		cfg.WorkspaceMode = true
	}
	m := NewModel(cfg)
	m.width = width
	m.height = height
	daemon := &mocks.DockerDaemonMock{}
	m.daemon = daemon
	m.ready = true
	m.swarmMode = true

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

	// Size the tables before feeding them rows.
	m.resizeContentModels()

	// Populate every view from the mock daemon's fixed fixtures.
	m.containers.SetContainers(daemon.Containers(nil, 0))
	images, err := daemon.Images()
	if err != nil {
		t.Fatalf("mock images: %v", err)
	}
	m.images.SetImages(images)
	networks, err := daemon.Networks()
	if err != nil {
		t.Fatalf("mock networks: %v", err)
	}
	m.networks.SetNetworks(networks)
	volumes, err := daemon.VolumeList(context.Background())
	if err != nil {
		t.Fatalf("mock volumes: %v", err)
	}
	m.volumes.SetVolumes(volumes)
	usage, err := daemon.DiskUsage()
	if err != nil {
		t.Fatalf("mock disk usage: %v", err)
	}
	m.diskUsage.SetUsage(usage)
	nodes, err := daemon.Nodes()
	if err != nil {
		t.Fatalf("mock nodes: %v", err)
	}
	m.nodes.SetNodes(nodes)
	services, err := daemon.Services()
	if err != nil {
		t.Fatalf("mock services: %v", err)
	}
	m.services.SetServices(services)
	stacks, err := daemon.Stacks()
	if err != nil {
		t.Fatalf("mock stacks: %v", err)
	}
	m.stacks.SetStacks(stacks)
	m.tasks.SetTasks(goldenTasks(), "Tasks")
	m.composeProjects.SetProjects(daemon.ComposeProjectsWithServices())
	m.composeServices.SetServices(daemon.ComposeServices("webapp"), nil, nil, "webapp")

	info, infoErr := daemon.Info()
	ver, verErr := daemon.Version()
	m.header = appui.NewHeaderModel(m.daemon, width)
	m.header.SetDockerInfo(info, infoErr, ver, verErr)

	for _, s := range goldenStats() {
		result, _ := m.Update(appui.MonitorStatsMsg{CID: s.ID, Stats: s})
		m = result.(model)
	}
	m.monitor.FlushTable()

	m.resizeContentModels()
	return m
}

func TestGoldenViews(t *testing.T) {
	cases := []struct {
		name      string
		view      viewMode
		workspace bool
		width     int
		height    int
	}{
		{name: "main", view: Main, width: 120, height: 40},
		{name: "main_narrow", view: Main, width: 80, height: 24},
		{name: "images", view: Images, width: 120, height: 40},
		{name: "networks", view: Networks, width: 120, height: 40},
		{name: "volumes", view: Volumes, width: 120, height: 40},
		{name: "disk_usage", view: DiskUsage, width: 120, height: 40},
		{name: "monitor", view: Monitor, width: 120, height: 40},
		{name: "nodes", view: Nodes, width: 120, height: 40},
		{name: "services", view: Services, width: 120, height: 40},
		{name: "stacks", view: Stacks, width: 120, height: 40},
		{name: "tasks", view: Tasks, width: 120, height: 40},
		{name: "compose_projects", view: ComposeProjects, width: 120, height: 40},
		{name: "compose_services", view: ComposeServices, width: 120, height: 40},
		{name: "workspace_main", view: Main, workspace: true, width: 120, height: 40},
		{name: "workspace_monitor", view: Monitor, workspace: true, width: 120, height: 40},
		{name: "workspace_main_compact", view: Main, workspace: true, width: 90, height: 30},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newGoldenModel(t, tc.workspace, tc.width, tc.height)
			m.view = tc.view
			m.resizeContentModels()

			rendered := m.renderMainScreen()
			if again := m.renderMainScreen(); again != rendered {
				t.Fatal("view rendering is nondeterministic: two consecutive renders differ")
			}
			got := normalizeGoldenView(rendered)

			path := filepath.Join("testdata", "golden", tc.name+".golden")
			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("missing golden file %s (generate with: go test ./app -run TestGoldenViews -update): %v", path, err)
			}
			if got != string(want) {
				gotPath := path + ".got"
				_ = os.WriteFile(gotPath, []byte(got), 0o644)
				t.Errorf("rendered view diverges from %s\nactual output written to %s\nif the change is intentional, regenerate with: go test ./app -run TestGoldenViews -update\nfirst differing line: %s",
					path, gotPath, firstGoldenDiff(string(want), got))
			}
		})
	}
}

func firstGoldenDiff(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			return fmt.Sprintf("line %d:\n  want: %q\n  got:  %q", i+1, w, g)
		}
	}
	return "(no line diff; length or trailing content differs)"
}
