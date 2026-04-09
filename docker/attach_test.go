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

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type attachAPIClientMock struct {
	client.APIClient
	inspect   container.InspectResponse
	insErr    error
	attachErr error
	output    string

	lastAttachOpts client.ContainerAttachOptions
}

func (m *attachAPIClientMock) ContainerInspect(context.Context, string, client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{Container: m.inspect}, m.insErr
}

func (m *attachAPIClientMock) ContainerAttach(ctx context.Context, containerID string, options client.ContainerAttachOptions) (client.ContainerAttachResult, error) {
	m.lastAttachOpts = options
	if m.attachErr != nil {
		return client.ContainerAttachResult{}, m.attachErr
	}

	clientConn, serverConn := net.Pipe()
	go func() {
		if m.output != "" {
			_, _ = io.WriteString(serverConn, m.output)
		}
		_ = serverConn.Close()
	}()

	return client.ContainerAttachResult{
		HijackedResponse: client.NewHijackedResponse(clientConn, "application/vnd.docker.raw-stream"),
	}, nil
}

func TestDockerDaemon_AttachInteractive_NonRunning(t *testing.T) {
	daemon := &DockerDaemon{client: &attachAPIClientMock{
		inspect: container.InspectResponse{
			State:  &container.State{Running: false},
			Config: &container.Config{Tty: true},
		},
	}}

	err := daemon.AttachInteractive(context.Background(), "abc123def456789", nil, io.Discard, io.Discard, "")
	if err == nil {
		t.Fatal("expected non-running container error")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerDaemon_AttachInteractive_AttachError(t *testing.T) {
	daemon := &DockerDaemon{client: &attachAPIClientMock{
		inspect: container.InspectResponse{
			State:  &container.State{Running: true},
			Config: &container.Config{Tty: true},
		},
		attachErr: errors.New("boom"),
	}}

	err := daemon.AttachInteractive(context.Background(), "abc123def456789", nil, io.Discard, io.Discard, "")
	if err == nil {
		t.Fatal("expected attach error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerDaemon_AttachInteractive_TTYSuccess(t *testing.T) {
	mockClient := &attachAPIClientMock{
		inspect: container.InspectResponse{
			State:  &container.State{Running: true},
			Config: &container.Config{Tty: true},
		},
		output: "hello from container\n",
	}
	daemon := &DockerDaemon{client: mockClient}

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
	if mockClient.lastAttachOpts.DetachKeys != defaultDetachKeys {
		t.Fatalf("expected default detach keys %q, got %q", defaultDetachKeys, mockClient.lastAttachOpts.DetachKeys)
	}
}

func TestDockerDaemon_AttachInteractive_FileStdinReturnsPromptly(t *testing.T) {
	// When stdin is an *os.File (as with Bubbletea's /dev/tty), attach
	// must return promptly after the output side closes — even though
	// nothing further will arrive on stdin.
	daemon := &DockerDaemon{client: &attachAPIClientMock{
		inspect: container.InspectResponse{
			State:  &container.State{Running: true},
			Config: &container.Config{Tty: true},
		},
		output: "hello\n",
	}}

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
