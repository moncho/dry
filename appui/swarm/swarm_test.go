package swarm

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moncho/dry/appui"
	"github.com/moncho/dry/docker"
)

func makeTestNodes(n int) []swarm.Node {
	nodes := make([]swarm.Node, n)
	for i := range n {
		nodes[i] = swarm.Node{
			ID: "node" + string(rune('a'+i)) + "1234567890",
			Spec: swarm.NodeSpec{
				Role:         swarm.NodeRoleWorker,
				Availability: swarm.NodeAvailabilityActive,
			},
			Status: swarm.NodeStatus{
				State: swarm.NodeStateReady,
			},
			Description: swarm.NodeDescription{
				Hostname: "host-" + string(rune('a'+i)),
				Resources: swarm.Resources{
					NanoCPUs:    4e9,
					MemoryBytes: 8 * 1024 * 1024 * 1024,
				},
			},
		}
	}
	return nodes
}

func makeTestServices(n int) []swarm.Service {
	services := make([]swarm.Service, n)
	for i := range n {
		replicas := uint64(i + 1)
		services[i] = swarm.Service{
			ID: "svc" + string(rune('a'+i)) + "1234567890",
			Spec: swarm.ServiceSpec{
				Annotations: swarm.Annotations{Name: "service-" + string(rune('a'+i))},
				Mode:        swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
				TaskTemplate: swarm.TaskSpec{
					ContainerSpec: &swarm.ContainerSpec{Image: "nginx:latest"},
				},
			},
		}
	}
	return services
}

func makeTestStacks(n int) []docker.Stack {
	stacks := make([]docker.Stack, n)
	for i := range n {
		stacks[i] = docker.Stack{
			Name:     "stack-" + string(rune('a'+i)),
			Services: i + 1,
			Networks: 1,
			Configs:  0,
			Secrets:  0,
		}
	}
	return stacks
}

func TestNodesModel_SetAndSelect(t *testing.T) {
	m := NewNodesModel()
	m.SetSize(120, 30)

	nodes := makeTestNodes(3)
	m.SetNodes(nodes)

	sel := m.SelectedNode()
	if sel == nil {
		t.Fatal("expected non-nil selected node")
	}
	if sel.ID != nodes[0].ID {
		t.Fatalf("expected first node ID %q, got %q", nodes[0].ID, sel.ID)
	}

	// Navigate down
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	sel = m.SelectedNode()
	if sel != nil && sel.ID != nodes[1].ID {
		t.Fatalf("expected second node after j, got %q", sel.ID)
	}
}

func TestNodesModel_EmptySelected(t *testing.T) {
	m := NewNodesModel()
	m.SetSize(120, 30)

	if m.SelectedNode() != nil {
		t.Fatal("expected nil selected node for empty model")
	}
}

func TestNodesModel_Sort(t *testing.T) {
	m := NewNodesModel()
	m.SetSize(120, 30)
	m.SetNodes(makeTestNodes(3))

	// F1 cycles sort
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	// No crash is sufficient
}

func TestNodesModel_ViewNotEmpty(t *testing.T) {
	m := NewNodesModel()
	m.SetSize(120, 30)
	m.SetNodes(makeTestNodes(2))

	v := m.View()
	if v == "" {
		t.Fatal("View() should not be empty")
	}
}

func TestServicesModel_SetAndSelect(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 30)

	services := makeTestServices(3)
	m.SetServices(services)

	sel := m.SelectedService()
	if sel == nil {
		t.Fatal("expected non-nil selected service")
	}
	if sel.ID != services[0].ID {
		t.Fatalf("expected first service ID %q, got %q", services[0].ID, sel.ID)
	}
}

func TestServicesModel_EmptySelected(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 30)

	if m.SelectedService() != nil {
		t.Fatal("expected nil selected service for empty model")
	}
}

func TestServicesModel_Navigation(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 30)
	m.SetServices(makeTestServices(5))

	// Navigate to end
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G'})
	sel := m.SelectedService()
	if sel == nil || sel.ID != makeTestServices(5)[4].ID {
		t.Fatalf("expected last service after G")
	}

	// Navigate to beginning
	m, _ = m.Update(tea.KeyPressMsg{Code: 'g'})
	sel = m.SelectedService()
	if sel == nil || sel.ID != makeTestServices(5)[0].ID {
		t.Fatalf("expected first service after g")
	}
}

func TestServicesModel_NilReplicas(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 30)

	// Service with Replicated mode but nil Replicas pointer
	services := []swarm.Service{
		{
			ID: "svc-nil-replicas",
			Spec: swarm.ServiceSpec{
				Annotations:  swarm.Annotations{Name: "no-replicas"},
				Mode:         swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: nil}},
				TaskTemplate: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: "nginx:latest"}},
			},
		},
	}
	// Should not panic
	m.SetServices(services)

	sel := m.SelectedService()
	if sel == nil {
		t.Fatal("expected non-nil selected service")
	}
}

func TestServicesModel_NilContainerSpec(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 30)

	// Service with nil ContainerSpec (e.g., plugin-based task)
	replicas := uint64(1)
	services := []swarm.Service{
		{
			ID: "svc-nil-container",
			Spec: swarm.ServiceSpec{
				Annotations:  swarm.Annotations{Name: "no-container-spec"},
				Mode:         swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}},
				TaskTemplate: swarm.TaskSpec{ContainerSpec: nil},
			},
		},
	}
	// Should not panic
	m.SetServices(services)

	sel := m.SelectedService()
	if sel == nil {
		t.Fatal("expected non-nil selected service")
	}
}

func TestServicesModel_GlobalMode(t *testing.T) {
	m := NewServicesModel()
	m.SetSize(120, 30)

	// Global mode service (Replicated is nil)
	services := []swarm.Service{
		{
			ID: "svc-global",
			Spec: swarm.ServiceSpec{
				Annotations:  swarm.Annotations{Name: "global-svc"},
				Mode:         swarm.ServiceMode{Replicated: nil},
				TaskTemplate: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: "nginx:latest"}},
			},
		},
	}
	// Should not panic
	m.SetServices(services)

	sel := m.SelectedService()
	if sel == nil {
		t.Fatal("expected non-nil selected service")
	}
}

func TestStacksModel_SetAndSelect(t *testing.T) {
	m := NewStacksModel()
	m.SetSize(120, 30)

	stacks := makeTestStacks(3)
	m.SetStacks(stacks)

	sel := m.SelectedStack()
	if sel == nil {
		t.Fatal("expected non-nil selected stack")
	}
	if sel.Name != stacks[0].Name {
		t.Fatalf("expected first stack %q, got %q", stacks[0].Name, sel.Name)
	}
}

func TestStacksModel_EmptySelected(t *testing.T) {
	m := NewStacksModel()
	m.SetSize(120, 30)

	if m.SelectedStack() != nil {
		t.Fatal("expected nil selected stack for empty model")
	}
}

func TestStacksModel_ViewNotEmpty(t *testing.T) {
	m := NewStacksModel()
	m.SetSize(120, 30)
	m.SetStacks(makeTestStacks(2))

	v := m.View()
	if v == "" {
		t.Fatal("View() should not be empty")
	}
}

func TestTasksModel_View(t *testing.T) {
	m := NewTasksModel()
	m.SetSize(120, 30)

	v := m.View()
	if v == "" {
		t.Fatal("View() should not be empty")
	}
}

// TestSwarmViews_EveryColumnKeepsItsGutterAtEveryWidth sweeps the swarm
// column sets the way appui sweeps its own: at every width, the cell before
// each column boundary must be a space, or one column's text runs into the
// next one's. The sets live in this package, so the sweep does too.
func TestSwarmViews_EveryColumnKeepsItsGutterAtEveryWidth(t *testing.T) {
	appui.InitStyles()
	nodes := NewNodesModel()
	services := NewServicesModel()
	stacks := NewStacksModel()
	tasks := NewTasksModel()
	sets := map[string]appui.TableModel{
		"nodes":    nodes.table,
		"services": services.table,
		"stacks":   stacks.table,
		"tasks":    tasks.table,
	}

	for name, tbl := range sets {
		t.Run(name, func(t *testing.T) {
			cells := make([]string, 8)
			for i := range cells {
				cells[i] = strings.Repeat(string(rune('a'+i)), 60)
			}
			for width := 20; width <= 210; width++ {
				table := tbl
				table.SetSize(width, 12)
				table.SetRows([]appui.TableRow{sweepRow(cells)})

				lines := strings.Split(table.View(), "\n")
				if len(lines) < 2 {
					t.Fatalf("width %d: expected a header and a row", width)
				}
				boundary := 0
				for i := range cells {
					boundary += table.ColumnWidth(i)
					if boundary == 0 || boundary >= width {
						break
					}
					if width := table.ColumnWidth(i); width < 2 {
						// One cell holds the ellipsis, with nothing left
						// for a gutter: the allocator prefers that to
						// overflowing the table. See
						// TestTableModel_FittingIsNeverTradedForSpacing.
						continue
					}
					for row, line := range lines[:2] {
						runes := []rune(ansi.Strip(line))
						if len(runes) < boundary {
							t.Fatalf("%s width %d: line %d is %d cells, expected %d",
								name, width, row, len(runes), boundary)
						}
						if runes[boundary-1] != ' ' {
							t.Fatalf("%s width %d: column %d butts the next on line %d:\n%s",
								name, width, i, row, ansi.Strip(line))
						}
					}
				}
			}
		})
	}
}

// sweepRow is a row of pre-set cells for the width sweep.
type sweepRow []string

func (r sweepRow) Columns() []string { return r }
func (r sweepRow) ID() string        { return "sweep" }
