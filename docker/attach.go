package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

const defaultDetachKeys = "ctrl-p,ctrl-q"

// AttachInteractive attaches stdin/stdout/stderr to a running container.
// It blocks until the attach session ends (detach, process exit, or error).
func (daemon *DockerDaemon) AttachInteractive(ctx context.Context, id string, stdin io.Reader, stdout, stderr io.Writer, detachKeys string) error {
	inspect, err := daemon.Inspect(id)
	if err != nil {
		return fmt.Errorf("attach inspect %s: %w", shortContainerID(id), err)
	}
	if inspect.State == nil || !inspect.State.Running {
		return fmt.Errorf("container %s is not running", shortContainerID(id))
	}

	if detachKeys == "" {
		detachKeys = defaultDetachKeys
	}

	attach, err := daemon.client.ContainerAttach(ctx, id, container.AttachOptions{
		Stream:     true,
		Stdin:      true,
		Stdout:     true,
		Stderr:     true,
		DetachKeys: detachKeys,
		Logs:       false,
	})
	if err != nil {
		return fmt.Errorf("attach %s: %w", shortContainerID(id), err)
	}
	defer attach.Close()

	tty := inspect.Config != nil && inspect.Config.Tty
	if tty {
		return streamInteractive(attach, stdin, stdout, true)
	}

	// Non-TTY: demultiplex stdout/stderr via stdcopy.
	return streamInteractiveStdcopy(attach, stdin, stdout, stderr)
}

// streamInteractive handles bidirectional stream copy for TTY sessions
// (attach and exec). When forwardStdin is true, stdin is copied to the
// connection in a background goroutine. Set forwardStdin to false for
// non-interactive exec commands so that stdin remains available to the
// caller after this function returns.
func streamInteractive(attach types.HijackedResponse, stdin io.Reader, stdout io.Writer, forwardStdin bool) error {
	// Copy stdin → connection in background.
	// When the output side finishes, closing the connection causes a
	// write error that terminates this goroutine.
	if forwardStdin && stdin != nil {
		go func() {
			_, _ = io.Copy(attach.Conn, stdin)
			_ = attach.CloseWrite()
		}()
	}

	// Copy connection → stdout (blocks until detach/exit).
	_, outErr := io.Copy(stdout, attach.Reader)

	// Close the connection so the stdin goroutine gets a write error.
	_ = attach.CloseWrite()
	attach.Close()

	if outErr != nil && !isExpectedCloseError(outErr) {
		return outErr
	}
	return nil
}

// streamInteractiveStdcopy is like streamInteractive but demultiplexes
// Docker's multiplexed stream for non-TTY containers.
func streamInteractiveStdcopy(attach types.HijackedResponse, stdin io.Reader, stdout, stderr io.Writer) error {
	if stdin != nil {
		go func() {
			_, _ = io.Copy(attach.Conn, stdin)
			_ = attach.CloseWrite()
		}()
	}

	_, outErr := stdcopy.StdCopy(stdout, stderr, attach.Reader)

	_ = attach.CloseWrite()
	attach.Close()

	if outErr != nil && !isExpectedCloseError(outErr) {
		return outErr
	}
	return nil
}

func isExpectedCloseError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "read canceled")
}

func shortContainerID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
