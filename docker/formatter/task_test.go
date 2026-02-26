package formatter

import (
	"io"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/moncho/dry/docker"
)

type mockSwarmAPI struct{}

func (m *mockSwarmAPI) Node(id string) (*swarm.Node, error)                        { return nil, nil }
func (m *mockSwarmAPI) NodeChangeAvailability(string, swarm.NodeAvailability) error { return nil }
func (m *mockSwarmAPI) Nodes() ([]swarm.Node, error)                               { return nil, nil }
func (m *mockSwarmAPI) NodeTasks(string) ([]swarm.Task, error)                     { return nil, nil }
func (m *mockSwarmAPI) ResolveNode(id string) (string, error)                      { return "node-1", nil }
func (m *mockSwarmAPI) ResolveService(id string) (string, error)                   { return "my-service", nil }
func (m *mockSwarmAPI) Service(id string) (*swarm.Service, error)                  { return nil, nil }
func (m *mockSwarmAPI) ServiceLogs(string, string, bool) (io.ReadCloser, error)    { return nil, nil }
func (m *mockSwarmAPI) Services() ([]swarm.Service, error)                         { return nil, nil }
func (m *mockSwarmAPI) ServiceRemove(string) error                                 { return nil }
func (m *mockSwarmAPI) ServiceScale(string, uint64) error                          { return nil }
func (m *mockSwarmAPI) ServiceTasks(...string) ([]swarm.Task, error)               { return nil, nil }
func (m *mockSwarmAPI) ServiceUpdate(string) error                                 { return nil }
func (m *mockSwarmAPI) Stacks() ([]docker.Stack, error)                            { return nil, nil }
func (m *mockSwarmAPI) StackConfigs(string) ([]swarm.Config, error)                { return nil, nil }
func (m *mockSwarmAPI) StackNetworks(string) ([]types.NetworkResource, error)      { return nil, nil }
func (m *mockSwarmAPI) StackSecrets(string) ([]swarm.Secret, error)                { return nil, nil }
func (m *mockSwarmAPI) StackRemove(string) error                                   { return nil }
func (m *mockSwarmAPI) StackTasks(string) ([]swarm.Task, error)                    { return nil, nil }
func (m *mockSwarmAPI) Task(string) (swarm.Task, error)                            { return swarm.Task{}, nil }

func TestTaskStringer_NilContainerSpec(t *testing.T) {
	task := swarm.Task{
		ID:        "task123",
		ServiceID: "svc456",
		Spec: swarm.TaskSpec{
			ContainerSpec: nil, // e.g., plugin-based task
		},
	}

	ts := NewTaskStringer(&mockSwarmAPI{}, task, true)

	// Should return empty string, not panic
	img := ts.Image()
	if img != "" {
		t.Fatalf("expected empty image for nil ContainerSpec, got %q", img)
	}
}

func TestTaskStringer_WithContainerSpec(t *testing.T) {
	task := swarm.Task{
		ID:        "task789",
		ServiceID: "svc456",
		Spec: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image: "nginx:1.25",
			},
		},
	}

	ts := NewTaskStringer(&mockSwarmAPI{}, task, false)

	img := ts.Image()
	if img != "nginx:1.25" {
		t.Fatalf("expected 'nginx:1.25', got %q", img)
	}
}
