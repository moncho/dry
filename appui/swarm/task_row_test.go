package swarm

import (
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/moncho/dry/docker/formatter"
)

func TestTaskRow(t *testing.T) {
	task := swarm.Task{
		ID:          "task1",
		Annotations: swarm.Annotations{Name: "taskName"},
		Spec:        swarm.TaskSpec{},
	}

	ts := formatter.NewTaskStringer(task)
	row := NewTaskRow(task)

	if row == nil {
		t.Error("TaskRow was not created")
	}

	if row.ID.Text != task.ID {
		t.Errorf("TaskRow name is not 'test', got %s", row.ID.Text)
	}

	if row.Name.Text != task.Name {
		t.Errorf("TaskRow name is not 'test', got %s", row.Name.Text)
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
