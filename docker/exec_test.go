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

type execAPIClientMock struct {
	client.APIClient
	inspect   container.InspectResponse
	insErr    error
	createErr error
	attachErr error
	output    string

	lastExecOpts   client.ExecCreateOptions
	lastAttachOpts client.ExecAttachOptions
}

func (m *execAPIClientMock) ContainerInspect(ctx context.Context, ctr string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{Container: m.inspect}, m.insErr
}

func (m *execAPIClientMock) ExecCreate(ctx context.Context, containerID string, config client.ExecCreateOptions) (client.ExecCreateResult, error) {
	m.lastExecOpts = config
	if m.createErr != nil {
		return client.ExecCreateResult{}, m.createErr
	}
	return client.ExecCreateResult{ID: "exec-123"}, nil
}

func (m *execAPIClientMock) ExecAttach(ctx context.Context, _ string, config client.ExecAttachOptions) (client.ExecAttachResult, error) {
	m.lastAttachOpts = config
	if m.attachErr != nil {
		return client.ExecAttachResult{}, m.attachErr
	}
	clientConn, serverConn := net.Pipe()
	go func() {
		if m.output != "" {
			_, _ = io.WriteString(serverConn, m.output)
		}
		_ = serverConn.Close()
	}()
	return client.ExecAttachResult{
		HijackedResponse: client.NewHijackedResponse(clientConn, "application/vnd.docker.raw-stream"),
	}, nil
}

func TestDockerDaemon_ExecInteractive_NonRunning(t *testing.T) {
	daemon := &DockerDaemon{client: &execAPIClientMock{
		inspect: container.InspectResponse{
			State:  &container.State{Running: false},
			Config: &container.Config{Tty: true},
		},
	}}

	err := daemon.ExecInteractive(context.Background(), "abc123def456789", []string{"/bin/sh"}, nil, io.Discard, io.Discard, true)
	if err == nil {
		t.Fatal("expected non-running container error")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerDaemon_ExecInteractive_CreateError(t *testing.T) {
	daemon := &DockerDaemon{client: &execAPIClientMock{
		inspect: container.InspectResponse{
			State:  &container.State{Running: true},
			Config: &container.Config{Tty: true},
		},
		createErr: errors.New("create boom"),
	}}

	err := daemon.ExecInteractive(context.Background(), "abc123def456789", []string{"/bin/sh"}, nil, io.Discard, io.Discard, true)
	if err == nil {
		t.Fatal("expected exec create error")
	}
	if !strings.Contains(err.Error(), "create boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerDaemon_ExecInteractive_AttachError(t *testing.T) {
	daemon := &DockerDaemon{client: &execAPIClientMock{
		inspect: container.InspectResponse{
			State:  &container.State{Running: true},
			Config: &container.Config{Tty: true},
		},
		attachErr: errors.New("attach boom"),
	}}

	err := daemon.ExecInteractive(context.Background(), "abc123def456789", []string{"/bin/sh"}, nil, io.Discard, io.Discard, true)
	if err == nil {
		t.Fatal("expected exec attach error")
	}
	if !strings.Contains(err.Error(), "attach boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerDaemon_ExecInteractive_TTYSuccess(t *testing.T) {
	client := &execAPIClientMock{
		inspect: container.InspectResponse{
			State:  &container.State{Running: true},
			Config: &container.Config{Tty: true},
		},
		output: "exec output\n",
	}
	daemon := &DockerDaemon{client: client}

	var out bytes.Buffer
	err := daemon.ExecInteractive(
		context.Background(),
		"abc123def456789",
		[]string{"/bin/sh", "-c", "echo hello"},
		strings.NewReader(""),
		&out,
		io.Discard,
		true,
	)
	if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}
	if !strings.Contains(out.String(), "exec output") {
		t.Fatalf("unexpected output: %q", out.String())
	}
	if !client.lastExecOpts.TTY {
		t.Fatal("expected TTY to be true")
	}
	if !client.lastAttachOpts.TTY {
		t.Fatal("expected attach TTY to be true")
	}
	if len(client.lastExecOpts.Cmd) != 3 || client.lastExecOpts.Cmd[0] != "/bin/sh" {
		t.Fatalf("unexpected exec command: %v", client.lastExecOpts.Cmd)
	}
}

func TestDockerDaemon_ExecInteractive_AlwaysAllocatesTTY(t *testing.T) {
	// Exec should allocate a TTY even when the container was started without one.
	client := &execAPIClientMock{
		inspect: container.InspectResponse{
			State:  &container.State{Running: true},
			Config: &container.Config{Tty: false},
		},
		output: "no-tty container\n",
	}
	daemon := &DockerDaemon{client: client}

	var out bytes.Buffer
	err := daemon.ExecInteractive(
		context.Background(),
		"abc123def456789",
		[]string{"/bin/sh"},
		strings.NewReader(""),
		&out,
		io.Discard,
		true,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !client.lastExecOpts.TTY {
		t.Fatal("expected exec TTY to be true even for non-TTY container")
	}
	if !client.lastAttachOpts.TTY {
		t.Fatal("expected exec attach TTY to be true even for non-TTY container")
	}
}

func TestDockerDaemon_ExecInteractive_FileStdinReturnsPromptly(t *testing.T) {
	daemon := &DockerDaemon{client: &execAPIClientMock{
		inspect: container.InspectResponse{
			State:  &container.State{Running: true},
			Config: &container.Config{Tty: true},
		},
		output: "hello\n",
	}}

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = stdinR.Close() }()
	defer func() { _ = stdinW.Close() }()

	done := make(chan error, 1)
	go func() {
		done <- daemon.ExecInteractive(
			context.Background(),
			"abc123def456789",
			[]string{"/bin/sh"},
			stdinR,
			io.Discard,
			io.Discard,
			true,
		)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ExecInteractive did not return within 2s — stdin goroutine likely blocked")
	}
}
