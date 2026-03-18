package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
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
	execCfg := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          command,
	}
	execResp, err := daemon.client.ContainerExecCreate(ctx, id, execCfg)
	if err != nil {
		return fmt.Errorf("exec create %s: %w", shortContainerID(id), err)
	}

	attach, err := daemon.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		return fmt.Errorf("exec attach %s: %w", shortContainerID(id), err)
	}
	defer attach.Close()

	return streamInteractive(attach, stdin, stdout, forwardStdin)
}
