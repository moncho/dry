package swarm

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/docker/docker/api/types/swarm"
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
	if sel.ID != nodes[1].ID {
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
	if sel.ID != makeTestServices(5)[4].ID {
		t.Fatalf("expected last service after G")
	}

	// Navigate to beginning
	m, _ = m.Update(tea.KeyPressMsg{Code: 'g'})
	sel = m.SelectedService()
	if sel.ID != makeTestServices(5)[0].ID {
		t.Fatalf("expected first service after g")
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
