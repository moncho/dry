package swarm

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
)

func TestNodeRow(t *testing.T) {
	node := swarm.Node{
		Description: swarm.NodeDescription{
			Engine: swarm.EngineDescription{
				EngineVersion: "1.0",
			},
			Hostname: "test.local",
			Resources: swarm.Resources{
				NanoCPUs:    2 * 1e9,
				MemoryBytes: 1024 * 1024,
			},
		},
		Spec: swarm.NodeSpec{
			Role: swarm.NodeRoleManager,
		},
		Status: swarm.NodeStatus{
			State: swarm.NodeStateReady,
			Addr:  "6.6.6.6",
		},
	}

	row := NewNodeRow(node, nodeTableHeader())

	if row == nil {
		t.Error("NodeRow was not created")
	}

	if row.Name.Text != node.Description.Hostname {
		t.Errorf("NodeRow name is not 'test', got %s", row.Name.Text)
	}

	if row.Role.Text != string(swarm.NodeRoleManager) {
		t.Errorf("Unexpected NodeRow role, got %s, expected %s", row.Role.Text, swarm.NodeRoleManager)
	}
	if row.CPU.Text != "2" {
		t.Error("NodeRow does not have 2 CPUs")
	}
	if row.Memory.Text != "1MiB" {
		t.Errorf("NodeRow does not have 1 MiB of memory, got %s", row.Memory.Text)
	}
	if row.Engine.Text != "1.0" {
		t.Errorf("Unexpected NodeRow engine version, got %s, expected 1.0", row.Engine.Text)
	}
	if row.IPAddress.Text != node.Status.Addr {
		t.Errorf("Unexpected NodeRow IP address, got %s, expected %s", row.IPAddress.Text, node.Status.Addr)
	}
	if row.Status.Text != string(node.Status.State) {
		t.Errorf("Unexpected NodeRow state, got %s, expected %s", row.Status.Text, node.Status.State)
	}
	if row.Availability.Text != string(node.Spec.Availability) {
		t.Errorf("Unexpected NodeRow availability, got %s, expected %s", row.Availability.Text, node.Spec.Availability)
	}
}
