package docker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerAPI "github.com/docker/docker/client"
)

type attachAPIClientMock struct {
	dockerAPI.APIClient
	inspect   container.InspectResponse
	insErr    error
	attachErr error
	output    string

	lastAttachOpts container.AttachOptions
}

func (m *attachAPIClientMock) ContainerInspect(ctx context.Context, ctr string) (container.InspectResponse, error) {
	return m.inspect, m.insErr
}

func (m *attachAPIClientMock) ContainerAttach(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error) {
	m.lastAttachOpts = options
	if m.attachErr != nil {
		return types.HijackedResponse{}, m.attachErr
	}

	clientConn, serverConn := net.Pipe()
	go func() {
		if m.output != "" {
			_, _ = io.WriteString(serverConn, m.output)
		}
		_ = serverConn.Close()
	}()

	return types.NewHijackedResponse(clientConn, "application/vnd.docker.raw-stream"), nil
}

func TestDockerDaemon_AttachInteractive_NonRunning(t *testing.T) {
	client := &attachAPIClientMock{
		inspect: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				State: &container.State{Running: false},
			},
			Config: &container.Config{Tty: true},
		},
	}
	daemon := &DockerDaemon{client: client}

	err := daemon.AttachInteractive(context.Background(), "abc123def456789", nil, io.Discard, io.Discard, "")
	if err == nil {
		t.Fatal("expected non-running container error")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerDaemon_AttachInteractive_AttachError(t *testing.T) {
	client := &attachAPIClientMock{
		inspect: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				State: &container.State{Running: true},
			},
			Config: &container.Config{Tty: true},
		},
		attachErr: errors.New("boom"),
	}
	daemon := &DockerDaemon{client: client}

	err := daemon.AttachInteractive(context.Background(), "abc123def456789", nil, io.Discard, io.Discard, "")
	if err == nil {
		t.Fatal("expected attach error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerDaemon_AttachInteractive_TTYSuccess(t *testing.T) {
	client := &attachAPIClientMock{
		inspect: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				State: &container.State{Running: true},
			},
			Config: &container.Config{Tty: true},
		},
		output: "hello from container\n",
	}
	daemon := &DockerDaemon{client: client}

	var out bytes.Buffer
	err := daemon.AttachInteractive(
		context.Background(),
		"abc123def456789",
		strings.NewReader(""),
		&out,
		io.Discard,
		"",
	)
	if err != nil {
		t.Fatalf("unexpected attach error: %v", err)
	}
	if !strings.Contains(out.String(), "hello from container") {
		t.Fatalf("unexpected output: %q", out.String())
	}
	if client.lastAttachOpts.DetachKeys != defaultDetachKeys {
		t.Fatalf("expected default detach keys %q, got %q", defaultDetachKeys, client.lastAttachOpts.DetachKeys)
	}
}

func TestDockerDaemon_AttachInteractive_FileStdinReturnsPromptly(t *testing.T) {
	// When stdin is an *os.File (as with Bubbletea's /dev/tty), attach
	// must return promptly after the output side closes — even though
	// nothing further will arrive on stdin.
	client := &attachAPIClientMock{
		inspect: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				State: &container.State{Running: true},
			},
			Config: &container.Config{Tty: true},
		},
		output: "hello\n",
	}
	daemon := &DockerDaemon{client: client}

	// Use an os.Pipe so stdin is an *os.File that blocks on read.
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = stdinR.Close() }()
	defer func() { _ = stdinW.Close() }()

	done := make(chan error, 1)
	go func() {
		done <- daemon.AttachInteractive(
			context.Background(),
			"abc123def456789",
			stdinR,
			io.Discard,
			io.Discard,
			"",
		)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("AttachInteractive did not return within 2s — stdin goroutine likely blocked")
	}
}
