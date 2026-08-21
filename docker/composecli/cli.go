// Package composecli runs the Docker Compose CLI plugin. It builds command
// lines and manages processes; it holds no model types and does not import
// the rest of the docker package, so the parent can wire it in without an
// import cycle.
package composecli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// composeCancelGrace bounds how long Close waits, once it cancels a running
// compose command, for the child's output pipes to actually close before it
// forces them shut. Without this bound, a subprocess compose spawned that
// inherited the child's stdout — a credential helper, a buildx worker — and
// that outlives the killed child holds the pipe's write end open, and Close
// never returns: it is the package's only cancellation path, so a hang here
// freezes whatever UI is waiting on it. See exec.Cmd.WaitDelay.
const composeCancelGrace = 2 * time.Second

// Options configures how compose is invoked.
type Options struct {
	// Host is the Docker host to target, i.e. the value of DOCKER_HOST.
	// Empty means whatever the environment already says.
	Host string
	// Extra is appended to the child environment, mainly for tests.
	Extra []string
}

// CLI runs `docker compose` commands.
type CLI struct {
	opts    Options
	version string
}

// Detect verifies that the compose plugin is available and records its
// version. It runs one short-lived process and must be called from a command,
// never from the UI update path.
func Detect(opts Options) (*CLI, error) {
	c := &CLI{opts: opts}
	out, err := exec.Command("docker", "compose", "version").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker compose plugin not available: %w", err)
	}
	c.version = strings.TrimSpace(string(out))
	return c, nil
}

// Version is the version string reported by the plugin.
func (c *CLI) Version() string { return c.version }

// Project is the target of a compose command: a project name, the directory
// commands resolve relative paths against, and the files that define it.
type Project struct {
	Name       string
	WorkingDir string
	Files      []string
}

// args builds the argument list for a compose verb against a project.
func (c *CLI) args(p Project, verb ...string) []string {
	args := []string{"compose"}
	if p.WorkingDir != "" {
		args = append(args, "--project-directory", p.WorkingDir)
	}
	for _, f := range p.Files {
		args = append(args, "-f", f)
	}
	if p.Name != "" {
		args = append(args, "-p", p.Name)
	}
	return append(args, verb...)
}

// env is the child environment for every invocation.
func (c *CLI) env() []string {
	env := os.Environ()
	if c.opts.Host != "" {
		env = append(env, "DOCKER_HOST="+c.opts.Host)
	}
	return append(env, c.opts.Extra...)
}

// procReader streams a process's combined output. Closing it kills the
// process and reports its exit status. A termination this package itself
// caused — Close canceling a still-running command, or WaitDelay force-
// closing pipes a lingering grandchild held open — is reported as a nil
// error rather than as a failure; a termination the process chose on its
// own, including a genuine non-zero exit, is reported as-is. A concurrent
// reader — the expected usage, since cancellation is documented as "closing
// the returned reader" while another goroutine streams from it — sees a
// clean io.EOF when Close ends the stream, never the pipe's generic
// "closed" error, which would be indistinguishable from a real failure.
type procReader struct {
	pr     *io.PipeReader
	cancel context.CancelFunc
	done   chan struct{} // closed once the process has been waited on
	err    error         // the error to report; valid once done is closed

	once   sync.Once
	result error
}

// Read implements io.Reader. This package is the only thing that ever closes
// pr's read side (in Close, below), so a Read that unblocks with
// io.ErrClosedPipe is always Close doing its job, not a real stream failure
// — report it as a clean end-of-stream instead.
func (p *procReader) Read(b []byte) (int, error) {
	n, err := p.pr.Read(b)
	if errors.Is(err, io.ErrClosedPipe) {
		err = io.EOF
	}
	return n, err
}

func (p *procReader) Close() error {
	p.once.Do(func() {
		// Cancel first so a still-running compose does not block the wait,
		// then close the reader so a writer blocked on a full, undrained
		// pipe cannot block cmd.Wait() either.
		p.cancel()
		_ = p.pr.Close()
		<-p.done
		p.result = p.err
	})
	return p.result
}

// stream starts a compose command and returns a reader over its combined
// stdout and stderr. There is no timeout on the command itself: pulls and
// builds are legitimately slow. Closing the reader cancels the process,
// bounded by composeCancelGrace (see its doc comment).
func (c *CLI) stream(ctx context.Context, p Project, verb ...string) (io.ReadCloser, error) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "docker", c.args(p, verb...)...)
	cmd.Env = c.env()
	cmd.WaitDelay = composeCancelGrace
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	pReader := &procReader{pr: pr, cancel: cancel, done: make(chan struct{})}
	go func() {
		pReader.err = classifyExit(cmd.Wait())
		_ = pw.Close() // unblocks any reader still waiting on EOF
		close(pReader.done)
	}()
	return pReader, nil
}

// classifyExit turns cmd.Wait's error into what Close should report. It
// suppresses two shapes that reflect this package ending the process itself
// rather than the process failing on its own:
//
//   - the process was terminated by a signal (ExitCode() == -1) — whether
//     from our own cancellation or from a WaitDelay-forced kill of a child
//     that ignored it;
//   - exec.ErrWaitDelay, which exec.Cmd only returns when the command
//     itself exited successfully but WaitDelay had to force-close pipes a
//     lingering grandchild (a credential helper, say) was still holding
//     open — not a failure of compose.
//
// Everything else, including a genuine non-zero exit code, is reported
// as-is. This classifies on the process's actual, final exit state rather
// than on whether or when Close happened to run, so it can't be confused by
// a Close that races a natural exit the way checking the context's
// cancellation cause could be.
func classifyExit(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, exec.ErrWaitDelay) {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == -1 {
		return nil
	}
	return err
}

// output runs a compose command to completion and returns its stdout.
func (c *CLI) output(ctx context.Context, p Project, verb ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", c.args(p, verb...)...)
	cmd.Env = c.env()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%s", msg)
		}
		return "", err
	}
	return string(out), nil
}

// Up creates and starts the project, or only the named services.
func (c *CLI) Up(ctx context.Context, p Project, services ...string) (io.ReadCloser, error) {
	return c.stream(ctx, p, append([]string{"up", "-d"}, services...)...)
}

// Down stops and removes the project's containers, networks, and default volumes.
func (c *CLI) Down(ctx context.Context, p Project) (io.ReadCloser, error) {
	return c.stream(ctx, p, "down")
}

// Recreate forces recreation of one service even when its config is unchanged.
func (c *CLI) Recreate(ctx context.Context, p Project, service string) (io.ReadCloser, error) {
	return c.stream(ctx, p, "up", "-d", "--force-recreate", service)
}

// Config returns the project's rendered configuration.
func (c *CLI) Config(ctx context.Context, p Project) (string, error) {
	return c.output(ctx, p, "config")
}

// ConfigHashes returns the per-service config hash of the project's files.
// These are comparable to each container's com.docker.compose.config-hash
// label, which is how compose itself decides whether to recreate a container.
func (c *CLI) ConfigHashes(ctx context.Context, p Project) (map[string]string, error) {
	out, err := c.output(ctx, p, "config", "--hash=*")
	if err != nil {
		return nil, err
	}
	return parseConfigHashes(out), nil
}

// ResolveProject asks compose for the real project defined by the given files,
// which respects a `name:` key in the file rather than guessing from the
// directory name.
func (c *CLI) ResolveProject(ctx context.Context, dir string, files []string) (Project, error) {
	probe := Project{WorkingDir: dir, Files: files}
	out, err := c.output(ctx, probe, "config", "--format", "json")
	if err != nil {
		return Project{}, err
	}
	var parsed struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return Project{}, fmt.Errorf("parse compose config: %w", err)
	}
	if parsed.Name == "" {
		return Project{}, fmt.Errorf("compose config reported no project name")
	}
	probe.Name = parsed.Name
	return probe, nil
}
