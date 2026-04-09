package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/moby/moby/client"
)

// ExecInteractive creates an exec instance in a running container and
// attaches stdin/stdout/stderr to it. It blocks until the exec process
// exits or the context is cancelled.
func (daemon *DockerDaemon) ExecInteractive(ctx context.Context, id string, command []string, stdin io.Reader, stdout, stderr io.Writer, forwardStdin bool) error {
	inspect, err := daemon.Inspect(id)
	if err != nil {
		return fmt.Errorf("exec inspect %s: %w", shortContainerID(id), err)
	}
	if inspect.State == nil || !inspect.State.Running {
		return fmt.Errorf("container %s is not running", shortContainerID(id))
	}

	// Always allocate a TTY for exec — the user is running an interactive
	// command (like docker exec -it), regardless of how the container was started.
	execResp, err := daemon.client.ExecCreate(ctx, id, client.ExecCreateOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		Cmd:          command,
	})
	if err != nil {
		return fmt.Errorf("exec create %s: %w", shortContainerID(id), err)
	}

	attach, err := daemon.client.ExecAttach(ctx, execResp.ID, client.ExecAttachOptions{
		TTY: true,
	})
	if err != nil {
		return fmt.Errorf("exec attach %s: %w", shortContainerID(id), err)
	}
	defer attach.Close()

	return streamInteractive(attach.HijackedResponse, stdin, stdout, forwardStdin)
}
