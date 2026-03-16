package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

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

	// Copy stdin to the attach connection in a goroutine.
	// When the output side finishes (container exits or detach), closing
	// the attach connection will unblock the stdin copy.
	var stdinDone chan error
	if stdin != nil {
		stdinDone = make(chan error, 1)
		go func() {
			_, copyErr := io.Copy(attach.Conn, stdin)
			_ = attach.CloseWrite()
			stdinDone <- copyErr
		}()
	}

	// Copy container output to stdout/stderr (blocks until detach/exit).
	var outErr error
	if inspect.Config != nil && inspect.Config.Tty {
		_, outErr = io.Copy(stdout, attach.Reader)
	} else {
		_, outErr = stdcopy.StdCopy(stdout, stderr, attach.Reader)
	}

	// Output ended — close the connection to unblock stdin goroutine.
	_ = attach.CloseWrite()
	attach.Close()

	if stdinDone != nil {
		// The stdin goroutine may be blocked on stdin.Read (waiting for
		// user input). Setting a read deadline on the file unblocks it
		// immediately so we don't hang until the user presses a key.
		if f, ok := stdin.(*os.File); ok {
			_ = f.SetReadDeadline(time.Now())
		}
		stdinErr := <-stdinDone
		// Clear the deadline so Bubbletea can reuse the fd normally.
		if f, ok := stdin.(*os.File); ok {
			_ = f.SetReadDeadline(time.Time{})
		}
		if stdinErr != nil && !isExpectedAttachCloseError(stdinErr) && outErr == nil {
			return stdinErr
		}
	}

	if outErr != nil && !isExpectedAttachCloseError(outErr) {
		return outErr
	}
	return nil
}

func isExpectedAttachCloseError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	// Some platforms wrap closed connection errors as plain strings.
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
