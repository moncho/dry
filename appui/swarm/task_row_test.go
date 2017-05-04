package swarm

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/moncho/dry/docker/formatter"
	"github.com/moncho/dry/mocks"
)

func TestTaskRow(t *testing.T) {
	task := swarm.Task{
		ID:        "task1",
		ServiceID: "1",
		Spec:      swarm.TaskSpec{},
		Slot:      0,
		NodeID:    "1",
	}
	client := &mocks.SwarmDockerDaemon{}
	ts := formatter.NewTaskStringer(client, task, true)
	row := NewTaskRow(client, task, taskTableHeader())

	if row == nil {
		t.Error("TaskRow was not created")
	}

	if row.ID.Text != task.ID {
		t.Errorf("TaskRow ID is not 'task1', got %s", row.ID.Text)
	}

	if row.Name.Text != "Service1.1" {
		t.Errorf("TaskRow name is not %s, got %s", "Service1.1", row.Name.Text)
	}

	if row.Image.Text != ts.Image() {
		t.Errorf("Unexpected TaskRow image, got %s, expected %s", row.Image.Text, ts.Image())
	}
	if row.Node.Text != ts.NodeID() {
		t.Errorf("Unexpected TaskRow node, got %s, expected %s", row.Node.Text, ts.NodeID())
	}
	if row.DesiredState.Text != ts.DesiredState() {
		t.Errorf("Unexpected TaskRow DesiredState, got %s, expected %s", row.DesiredState.Text, ts.DesiredState())
	}
	if row.CurrentState.Text != ts.CurrentState() {
		t.Errorf("Unexpected TaskRow CurrentState, got %s, expected %s", row.CurrentState.Text, ts.CurrentState())
	}
	if row.Error.Text != ts.Error() {
		t.Errorf("Unexpected TaskRow error, got %s, expected %s", row.Error.Text, ts.Error())
	}
	if row.Ports.Text != ts.Ports() {
		t.Errorf("Unexpected TaskRow ports, got %s, expected %s", row.Ports.Text, ts.Ports())

	}
}
